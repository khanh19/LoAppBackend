package stamps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
)

//encore:api auth method=POST path=/stamps/:id/photos/upload-url
func (s *Service) CreatePhotoUploadURL(ctx context.Context, id string, req *PhotoUploadURLRequest) (*PhotoUploadURLResponse, error) {
	userID, err := requireUserID()
	if err != nil {
		return nil, err
	}
	if err := s.assertStampOwner(ctx, userID, id); err != nil {
		return nil, err
	}
	if req == nil {
		req = &PhotoUploadURLRequest{}
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "other"
	}
	if label != "food_drink" && label != "space_vibe" && label != "other" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid photo label"}
	}
	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		contentType = "image/jpeg"
	}

	ext := "jpg"
	switch contentType {
	case "image/png":
		ext = "png"
	case "image/webp":
		ext = "webp"
	}

	path := fmt.Sprintf("stamps/%s/%d.%s", id, time.Now().UnixNano(), ext)
	base := strings.TrimRight(secrets.SupabaseURL, "/")
	if base == "" {
		base = "https://xofbxjbrzpsocnjdsmmk.supabase.co"
	}
	if strings.TrimSpace(secrets.SupabaseStorageServiceKey) == "" {
		// Local/dev without storage key: return path for client-side wiring.
		return &PhotoUploadURLResponse{
			UploadURL:   fmt.Sprintf("%s/storage/v1/object/LoAppBucket/%s", base, path),
			StoragePath: path,
		}, nil
	}

	signed, err := createSupabaseSignedUpload(ctx, base, secrets.SupabaseStorageServiceKey, "LoAppBucket", path)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create signed upload url")
	}
	return &PhotoUploadURLResponse{
		UploadURL:   signed.URL,
		StoragePath: path,
		Token:       signed.Token,
	}, nil
}

//encore:api auth method=POST path=/stamps/:id/photos
func (s *Service) ConfirmPhoto(ctx context.Context, id string, req *ConfirmPhotoRequest) (*ConfirmPhotoResponse, error) {
	userID, err := requireUserID()
	if err != nil {
		return nil, err
	}
	if err := s.assertStampOwner(ctx, userID, id); err != nil {
		return nil, err
	}
	if req == nil || strings.TrimSpace(req.StoragePath) == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "storage_path is required"}
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "other"
	}
	if label != "food_drink" && label != "space_vibe" && label != "other" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid photo label"}
	}
	stampUUID, err := uuidFromString(id)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid stamp id"}
	}
	row, err := dbgen.New(s.db).InsertStampPhoto(ctx, dbgen.InsertStampPhotoParams{
		StampID:     stampUUID,
		StoragePath: strings.TrimSpace(req.StoragePath),
		Label:       label,
		SortOrder:   int32(req.SortOrder),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to save photo")
	}
	return &ConfirmPhotoResponse{
		PhotoID:     row.ID,
		StoragePath: row.StoragePath,
		Label:       row.Label,
	}, nil
}

func (s *Service) assertStampOwner(ctx context.Context, userID, stampID string) error {
	stampUUID, err := uuidFromString(stampID)
	if err != nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid stamp id"}
	}
	stamp, err := dbgen.New(s.db).GetStampByID(ctx, stampUUID)
	if err != nil {
		return &errs.Error{Code: errs.NotFound, Message: "stamp not found"}
	}
	if stamp.UserID != userID {
		return &errs.Error{Code: errs.PermissionDenied, Message: "stamp does not belong to user"}
	}
	return nil
}

type signedUploadResult struct {
	URL   string
	Token string
}

func createSupabaseSignedUpload(ctx context.Context, baseURL, serviceKey, bucket, path string) (*signedUploadResult, error) {
	url := fmt.Sprintf("%s/storage/v1/object/upload/sign/%s/%s", strings.TrimRight(baseURL, "/"), bucket, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+serviceKey)
	req.Header.Set("apikey", serviceKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("supabase storage %d: %s", res.StatusCode, string(body))
	}
	var parsed struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	uploadURL := parsed.URL
	if !strings.HasPrefix(uploadURL, "http") {
		uploadURL = strings.TrimRight(baseURL, "/") + uploadURL
	}
	return &signedUploadResult{URL: uploadURL, Token: parsed.Token}, nil
}
