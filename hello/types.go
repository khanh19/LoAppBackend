package hello

type Response struct {
	Message string
}

// DBPingResponse is the result of DBPing.
type DBPingResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}
