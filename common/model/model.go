package model

type Request struct {
	StartTime string `json:"started,omitempty"`
	Host      string `json:"host,omitempty"`

	RemoteAddr string `json:"remote_addr,omitempty"`
	LocalAddr  string `json:"local_addr,omitempty"`

	Method     string            `json:"method,omitempty"`
	Header     map[string]string `json:"header,omitempty"`
	URI        string            `json:"uri,omitempty"`
	Path       string            `json:"path,omitempty"`
	QueryArgs  map[string]string `json:"query_args,omitempty"`
	BodyLength int               `json:"body_length"`
	Body       string            `json:"body,omitempty"`
	User       string            `json:"user,omitempty"`
}

type Response struct {
	Cached bool `json:"cached"`

	LocalAddr  string `json:"local_addr,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`

	Header          map[string]string `json:"header,omitempty"`
	StatusCode      int               `json:"status_code"`
	BodyLength      int               `json:"body_length"`
	Body            string            `json:"body,omitempty"`
	ElapsedTimeInMs float32           `json:"elapsed"`
}

type DataFlow struct {
	From    string   `json:"from"`
	Relay   string   `json:"relay"`
	To      []string `json:"to"`
	Process []string `json:"process"`
}

type HttpRequest struct {
	ID           uint64    `json:"id,omitempty"`
	ConnTime     string    `json:"conn_time,omitempty"`
	LoggingTime  string    `json:"@timestamp,omitempty"`
	LocalIP      string    `json:"local_ip,omitempty"`
	RemoteIP     string    `json:"remote_ip,omitempty"`
	IsTLS        bool      `json:"tls"`
	TLSDidResume bool      `json:"tls_reuse,omitempty"`
	Request      *Request  `json:"request,omitempty"`
	Response     *Response `json:"response,omitempty"`
	DataFlow     *DataFlow `json:"flow,omitempty"`
	Elastic map[string]interface{} `json:"elastic,omitempty"`
}

//TODO elapsed time, domain(check), client->proxy->upstream make it works,
