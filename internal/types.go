package internal

const (
	APPROVED = "approved"
	REJECTED = "rejected"

	CfgPostTimeOut = "timeout"
	DefPostTimeOut = "15s"

	CfgExecPostTimeOut = "exec.timeout"
	DefExecPostTimeOut = "15s"

	CfgPort = "port"
	DefPort = 9000

	CfgExecBufferSize = "exec.bufferSize"
	DefExecBufferSize = 10

	CfgExecSampleInterval = "exec.sampleInterval"
	defExecSampleInterval = 100

	CfgExecServerUrl = "exec.serverUrl"
	defExecServerUrl = "http://localhost:8081"

	CfgAuthUser = "auth.user"
	DefAuthUser = "admin"

	CfgAuthPass = "auth.password"
	DefAuthPass = "password"
)

type Status string

type AppService interface {

}

type OrderRequest struct {
	Price uint `json:"price"`
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
	Error chan error
}

type FifoQueue interface {
    Insert(interface{}) error
    RemoveNValues() error
}