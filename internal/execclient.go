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
	stopCh  chan (struct{})
	cfg     *viper.Viper
	bOrders chan (orderWithChan)
}

func NewExecClient(cfg *viper.Viper, stopCh chan (struct{})) ExecClient {
	return &execClient{
		stopCh:  stopCh,
		cfg:     cfg,
		bOrders: make(chan (orderWithChan), cfg.GetUint(CfgExecBufferSize)),
	}
}

func (e *execClient) Run() {
	go e.runClient()
}

func (e *execClient) ProcessOrder(orq OrderRequest) OrderStatus {
	// create channels for status and error of the clients request
	ors := OrderStatus{
		Status: make(chan (Status), 1),
		Error:  make(chan (error), 1),
	}

	go func() {
		// push order to buffer without blocing the client
		e.bOrders <- orderWithChan{
			orq,
			ors,
		}
	}()

	return ors
}

func (e *execClient) runClient() {
	orws := []orderWithChan{}
	for {
		select {
		// in case the sub process is stoppped
		case <-e.stopCh:
			log.V(2).Info("exec client process was stopped")
			for _, orw := range orws {
				orw.Error <- fmt.Errorf("exec client process was stopped")
			}
			return
		// in case there is an order in the buffer
		case or := <-e.bOrders:
			orws = append(orws, or)
			if len(orws) == int(e.cfg.GetUint(CfgExecBufferSize)) {
				log.V(2).Info("fulfill orders")
				e.fulfillOrders(orws)
				orws = []orderWithChan{}
			}
		}
	}
}

func (e *execClient) fulfillOrders(orws []orderWithChan) {
	// query exec server for order status
	orps, err := e.queryExecServer(orws)

	// update error chan if error
	if err != nil {
		log.Errorf("can't fulfillOrders, %s", err)
		for _, orw := range orws {
			orw.Error <- fmt.Errorf("can't fulfillOrders, %s", err)
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

func (e *execClient) queryExecServer(orws []orderWithChan) ([]OrderResponse, error) {
	// encode the orders
	or := []OrderRequest{}
	for _, orw := range orws {
		or = append(or, orw.OrderRequest)
	}
	jsonData, err := json.Marshal(or)
	if err != nil {
		return nil, fmt.Errorf("can't encode orders, %s", err)
	}

	// POST request the server
	resp, err := http.Post(
		fmt.Sprintf("%s/v1/fulfillorder",
			e.cfg.GetString(CfgExecServerUrl)),
		"application/json",
		bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("can't POST exec server, %s", err)
	}
	defer resp.Body.Close()

	// check if empty
	if resp.Body == nil {
		return nil, fmt.Errorf("body of POST to exec server")
	}

	// decode to OrderResponse slice
	orp := []OrderResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&orp); err != nil {
		return nil, fmt.Errorf("can't decode response from exec server, %s", err)
	}
	return orp, nil
}
