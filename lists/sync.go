package lists

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const googleResolveWorkers = 3

func (s *Service) syncPlaceLists(ctx context.Context) (*SyncPlaceListsResponse, error) {
	file, err := loadSeedFile()
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to load seed lists")
	}

	q := dbgen.New(s.db)
	runID, err := q.CreateListSyncRun(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create sync run")
	}
	runUUID, err := uuidFromString(runID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "invalid sync run id")
	}

	var (
		added   int32
		skipped int32
		failed  int32
		details []SyncEntryResult
	)

	for listIndex, list := range file.Lists {
		listResults, listAdded, listSkipped, listFailed, err := s.syncSeedList(ctx, q, list, listIndex+1)
		if err != nil {
			finishErr := finishSyncRun(ctx, q, runUUID, "failed", added, skipped, failed+1, append(details, SyncEntryResult{
				ListSlug: list.ListID,
				Status:   "failed",
				Error:    err.Error(),
			}))
			if finishErr != nil {
				return nil, finishErr
			}
			return nil, errs.WrapCode(err, errs.Internal, "failed to sync list")
		}
		details = append(details, listResults...)
		added += listAdded
		skipped += listSkipped
		failed += listFailed
	}

	status := "completed"
	if failed > 0 {
		status = "completed"
	}
	if err := finishSyncRun(ctx, q, runUUID, status, added, skipped, failed, details); err != nil {
		return nil, err
	}

	return &SyncPlaceListsResponse{
		RunID:          runID,
		Status:         status,
		EntriesAdded:   added,
		EntriesSkipped: skipped,
		EntriesFailed:  failed,
	}, nil
}

func (s *Service) syncSeedList(ctx context.Context, q *dbgen.Queries, list seedList, sortOrder int) ([]SyncEntryResult, int32, int32, int32, error) {
	sourceHash := hashSeedList(list)

	existing, existingErr := q.GetPlaceListBySlug(ctx, list.ListID)
	if existingErr == nil && existing.SourceHash != nil && *existing.SourceHash == sourceHash {
		listUUID, parseErr := uuidFromString(existing.ID)
		if parseErr == nil {
			seedNames, namesErr := q.ListPlaceListEntrySeedNamesByListID(ctx, listUUID)
			if namesErr == nil && len(seedNames) >= len(list.Items) {
				return nil, 0, int32(len(list.Items)), 0, nil
			}
		}
	}
	if existingErr != nil && !errors.Is(existingErr, pgx.ErrNoRows) {
		return nil, 0, 0, 0, existingErr
	}

	city, err := resolveCity(ctx, q, list.Items)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	cityUUID, err := uuidFromString(city.ID)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	listID, err := q.UpsertPlaceList(ctx, dbgen.UpsertPlaceListParams{
		Slug:       list.ListID,
		Title:      list.Title,
		Category:   list.Category,
		Area:       list.Area,
		Occasions:  list.Occasions,
		CityID:     cityUUID,
		SortOrder:  int32(sortOrder),
		SourceHash: stringPtr(sourceHash),
	})
	if err != nil {
		return nil, 0, 0, 0, err
	}
	listUUID, err := uuidFromString(listID)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	lat, lng := cityCoords(city)
	existingNames := make(map[string]struct{})
	seedNames := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		seedNames = append(seedNames, item.SeedName)
	}
	rows, err := q.ListPlaceListEntrySeedNamesByListID(ctx, listUUID)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	for _, name := range rows {
		existingNames[name] = struct{}{}
	}

	type resolveJob struct {
		item seedItem
	}
	type resolveResult struct {
		item    seedItem
		placeID string
		status  string
		err     error
	}

	jobs := make(chan resolveJob)
	results := make(chan resolveResult)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for job := range jobs {
			if _, ok := existingNames[job.item.SeedName]; ok {
				results <- resolveResult{item: job.item, status: "skipped"}
				continue
			}

			placeID, status, resolveErr := s.resolvePlaceForItem(ctx, q, cityUUID, job.item, lat, lng)
			results <- resolveResult{
				item:    job.item,
				placeID: placeID,
				status:  status,
				err:     resolveErr,
			}
		}
	}

	for i := 0; i < googleResolveWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	go func() {
		for _, item := range list.Items {
			jobs <- resolveJob{item: item}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var (
		added   int32
		skipped int32
		failed  int32
		details []SyncEntryResult
	)

	for result := range results {
		entryResult := SyncEntryResult{
			ListSlug: list.ListID,
			SeedName: result.item.SeedName,
		}

		switch {
		case result.err != nil:
			entryResult.Status = "failed"
			entryResult.Error = result.err.Error()
			failed++
		case result.status == "skipped":
			entryResult.Status = "skipped"
			skipped++
		default:
			entryResult.Status = result.status
			entryResult.PlaceID = result.placeID
			if result.status == "added" {
				added++
			} else {
				skipped++
			}
		}
		details = append(details, entryResult)

		if result.err != nil || result.placeID == "" {
			continue
		}

		placeUUID, err := uuidFromString(result.placeID)
		if err != nil {
			continue
		}
		if err := q.UpsertPlaceListEntry(ctx, dbgen.UpsertPlaceListEntryParams{
			ListID:    listUUID,
			PlaceID:   placeUUID,
			SeedName:  result.item.SeedName,
			StopOrder: int32(result.item.StopOrder),
			Note:      result.item.Note,
		}); err != nil {
			return nil, added, skipped, failed, err
		}
	}

	for _, item := range list.Items {
		if _, ok := existingNames[item.SeedName]; !ok {
			continue
		}
		placeID, err := findExistingEntryPlaceID(ctx, q, listUUID, item.SeedName)
		if err != nil {
			continue
		}
		placeUUID, err := uuidFromString(placeID)
		if err != nil {
			continue
		}
		if err := q.UpsertPlaceListEntry(ctx, dbgen.UpsertPlaceListEntryParams{
			ListID:    listUUID,
			PlaceID:   placeUUID,
			SeedName:  item.SeedName,
			StopOrder: int32(item.StopOrder),
			Note:      item.Note,
		}); err != nil {
			return nil, added, skipped, failed, err
		}
	}

	if _, err := q.DeactivatePlaceListEntriesNotInSeedNames(ctx, dbgen.DeactivatePlaceListEntriesNotInSeedNamesParams{
		ListID:    listUUID,
		SeedNames: seedNames,
	}); err != nil {
		return nil, added, skipped, failed, err
	}

	return details, added, skipped, failed, nil
}

func (s *Service) resolvePlaceForItem(ctx context.Context, q *dbgen.Queries, cityID pgtype.UUID, item seedItem, lat, lng float64) (string, string, error) {
	placeID, err := q.FindPlaceByNameInCity(ctx, dbgen.FindPlaceByNameInCityParams{
		CityID: cityID,
		Name:   item.SeedName,
	})
	if err == nil {
		return placeID, "linked_existing", nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", "", err
	}

	resolved, err := s.google.resolvePlace(ctx, buildSearchQuery(item), lat, lng)
	if err != nil {
		return "", "", err
	}

	upsertedID, err := q.UpsertGooglePlace(ctx, dbgen.UpsertGooglePlaceParams{
		CityID:        cityID,
		GooglePlaceID: stringPtr(resolved.GooglePlaceID),
		Name:          resolved.Name,
		Neighborhood:  stringPtr(resolved.Neighborhood),
		Address:       stringPtr(resolved.Address),
		Latitude:      numericFromFloat(resolved.Latitude),
		Longitude:     numericFromFloat(resolved.Longitude),
		PriceLevel:    resolved.PriceLevel,
		RatingCached:  numericFromOptionalFloat(resolved.Rating),
		CoverImageUrl: stringPtr(resolved.CoverImageURL),
		Tags:          resolved.Tags,
	})
	if err != nil {
		return "", "", err
	}
	return upsertedID, "added", nil
}

func resolveCity(ctx context.Context, q *dbgen.Queries, items []seedItem) (dbgen.GetCityByHintRow, error) {
	for _, item := range items {
		if strings.TrimSpace(item.CityHint) == "" {
			continue
		}
		city, err := q.GetCityByHint(ctx, dbgen.GetCityByHintParams{
			CityHint:    item.CityHint,
			CityHintAlt: cityHintAlt(item.CityHint),
		})
		if err == nil {
			return city, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return dbgen.GetCityByHintRow{}, err
		}
	}
	return dbgen.GetCityByHintRow{}, errors.New("no city hint matched an active city")
}

func cityCoords(city dbgen.GetCityByHintRow) (float64, float64) {
	lat := numericToFloat(city.Latitude)
	lng := numericToFloat(city.Longitude)
	if lat == 0 && lng == 0 {
		switch city.Slug {
		case "hcmc":
			return 10.8231, 106.6297
		case "hanoi":
			return 21.0285, 105.8542
		case "danang":
			return 16.0544, 108.2022
		default:
			return 10.8231, 106.6297
		}
	}
	return lat, lng
}

func findExistingEntryPlaceID(ctx context.Context, q *dbgen.Queries, listID pgtype.UUID, seedName string) (string, error) {
	rows, err := q.ListPlaceListEntriesByListID(ctx, listID)
	if err != nil {
		return "", err
	}
	for _, row := range rows {
		if row.SeedName == seedName {
			return row.PlaceID, nil
		}
	}
	return "", pgx.ErrNoRows
}

func finishSyncRun(ctx context.Context, q *dbgen.Queries, runID pgtype.UUID, status string, added, skipped, failed int32, details []SyncEntryResult) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to encode sync details")
	}
	if err := q.FinishListSyncRun(ctx, dbgen.FinishListSyncRunParams{
		Status:         status,
		EntriesAdded:   added,
		EntriesSkipped: skipped,
		EntriesFailed:  failed,
		Details:        payload,
		ID:             runID,
	}); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to finish sync run")
	}
	return nil
}

func numericFromOptionalFloat(v *float64) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}
	return numericFromFloat(*v)
}
