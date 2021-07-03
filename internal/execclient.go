package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/viper"
	log "k8s.io/klog"
)

type orderWithChan struct {
	OrderRequest
	OrderStatus
}

type execClient struct {
	stopCh chan(struct{})
	cfg *viper.Viper
	bOrders chan(orderWithChan)
}

func NewExecClientCB(cfg *viper.Viper, stopCh chan(struct{})) ExecClient {
	return &execClient{
		stopCh: stopCh,
		cfg: cfg,
		bOrders: make(chan(orderWithChan), cfg.GetUint(CfgExecBufferSize)),
	}
}

func (e* execClient) Run() {
	go e.runClient()
}

func (e* execClient) ProcessOrder(orq OrderRequest) OrderStatus {
	// create channels for status and error of the clients request
	ors := OrderStatus{
		Status: make(chan(Status)),
		Error: make(chan(error)),
	}

	go func() {
		// push order to buffer without blocing the client 
		e.bOrders<- orderWithChan {
			orq,
			ors,
		}
	}()

	return ors
}

func (e* execClient) runClient() {
	orws := make([]orderWithChan, cap(e.bOrders))
	for {
		select {
			// in case the sub process is stoppped
			case <-e.stopCh:
				log.V(2).Info("exec client process was stopped")
				for _, orw := range orws {
					orw.Error <- fmt.Errorf("exec client process was stopped")
				}
				break
			// in case there is an order in the buffer
			case or := <-e.bOrders:
				orws = append(orws, or)
				if (len(orws) == cap(orws)) {
					log.V(2).Info("fulfill orders")
					e.fulfillOrders(orws)
					orws = make([]orderWithChan, cap(e.bOrders))
				}
		}
	}
}

func (e* execClient) fulfillOrders(orws []orderWithChan) {
	// query exec server for order status
	orps, err := e.queryExecServer(orws)
	
	// update error chan if error
	if err != nil {
		log.Errorf("can't fulfillOrders, %s", err)
		for _, orw := range orws {
			orw.Error <- err
		}
	}
	
	// update status channels if approved or rejected
	for _, orw := range orws {
		for _, orp := range orps {
			if orp.OrderRequest == orw.OrderRequest {
				orw.Status <- orp.Status
			}
		}
	}
}

func (e* execClient) queryExecServer(orws []orderWithChan) ([]OrderResponse, error) {
	// encode the orders
	b := new(bytes.Buffer)
	or := make([]OrderRequest, len(orws))
	for _, orw := range orws {
		or = append(or, orw.OrderRequest)
	}
	if err := json.NewEncoder(b).Encode(or); err != nil {
		return nil, fmt.Errorf("can't encode orders, ", err)
	}

	// POST request the server
	resp, err := http.Post(fmt.Sprintf("%s/fulfillorder", e.cfg.GetString(CfgExecServerUrl)), "application/json", b)
	if err != nil {
		return nil, fmt.Errorf("can't POST exec server, ", err)
	}
	defer resp.Body.Close()
	
	// check if empty
	if resp.Body == nil {
		return nil, fmt.Errorf("body of POST to exec server")
	}

	// decode to OrderResponse slice
	orp := []OrderResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&orp); err != nil {
		return nil, fmt.Errorf("can't decode response from exec server, ", err)
	}
	return orp, nil
}