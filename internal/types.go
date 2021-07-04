package internal

const (
	APPROVED = "approved"
	REJECTED = "rejected"

	CfgPort = "port"
	DefPort = 9000

	CfgPostTimeOut = "timeout"
	DefPostTimeOut = "15s"

	CfgExecPostTimeOut = "exec.timeout"
	DefExecPostTimeOut = "15s"

	CfgExecBufferSize = "exec.bufferSize"
	DefExecBufferSize = 10

	CfgExecSampleInterval = "exec.sampleInterval"
	DefExecSampleInterval = 100

	CfgExecServerUrl = "exec.serverUrl"
	DefExecServerUrl = "http://localhost:9081"

	CfgAuthUser = "auth.user"
	DefAuthUser = "admin"

	CfgAuthPass = "auth.password"
	DefAuthPass = "password"
)

type Status string

type OrderRequest struct {
	Price uint   `json:"price"`
	Order string `json:"order"`
}

type OrderResponse struct {
	OrderRequest
	Status Status `json:"status,omitempty"`
}

type ExecClient interface {
	ProcessOrder(or OrderRequest) OrderStatus
	Run()
}

type OrderStatus struct {
	Status chan Status
	Error  chan error
}
