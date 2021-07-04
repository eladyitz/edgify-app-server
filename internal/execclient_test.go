package internal_test

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/spf13/viper"
	. "github.com/i350641/edgify-app-server/internal"
)

var _ = Describe("exec client tests", func() {
	var cfg *viper.Viper
	var stopCh chan (struct{})
	var execClient ExecClient
	var reportServer *ghttp.Server
	var or1 OrderRequest
	var or2 OrderRequest
	var marshelOr []byte
	var reportRequest http.HandlerFunc

	BeforeEach(func() {
		cfg = viper.New()
		cfg.Set(CfgExecBufferSize, 2)
		stopCh = make(chan struct{}, 1)
		execClient = NewExecClient(cfg, stopCh)
		reportServer = ghttp.NewServer()
		or1 = OrderRequest{
			Price: 100,
			Order: "stam1",
		}
		or2 = OrderRequest{
			Price: 200,
			Order: "stam2",
		}

		or := []OrderRequest{or1, or2}
		marshelOr, _ = json.Marshal(or)

		reportRequest = ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", "/v1/fulfillorder"),
			ghttp.VerifyHeader(http.Header{"Content-Type": {"application/json"}}),
			ghttp.VerifyBody(marshelOr),
		)
	})

	AfterEach(func() {
		reportServer.Close()
	})

	Describe("run client in background", func() {
		It("stop if stopCh is triggered", func() {
			execClient.Run()
			stopCh <- struct{}{}
		})
		It("order process fails if exec server rejects", func() {
			reportServer.AppendHandlers(ghttp.RespondWith(401, ``), reportRequest)
			cfg.Set(CfgExecServerUrl, reportServer.URL())
			execClient.Run()
			ors1 := execClient.ProcessOrder(or1)
			ors2 := execClient.ProcessOrder(or2)
			time.Sleep(4 * time.Second)
			stopCh <- struct{}{}
			Expect(len(ors1.Status)).To(Equal(0))
			Expect(len(ors2.Status)).To(Equal(0))
			Expect((<-ors1.Error).Error()).To(ContainSubstring("can't fulfillOrders,"))
			Expect((<-ors2.Error).Error()).To(ContainSubstring("can't fulfillOrders,"))
		})
		It("order process fails if orders in queue (under buffer size) but process terminated", func() {
			cfg.Set(CfgExecBufferSize, 3)
			execClient.Run()
			ors1 := execClient.ProcessOrder(or1)
			ors2 := execClient.ProcessOrder(or2)
			time.Sleep(4 * time.Second)
			stopCh <- struct{}{}
			Expect(len(ors1.Status)).To(Equal(0))
			Expect(len(ors2.Status)).To(Equal(0))
			Expect((<-ors1.Error).Error()).To(ContainSubstring("exec client process was stopped"))
			Expect((<-ors2.Error).Error()).To(ContainSubstring("exec client process was stopped"))
		})
		It("order proces succedes", func() {
			ors := []OrderResponse{{or1, Status(APPROVED)}, {or2, Status(REJECTED)}}
			marshelOrs, _ := json.Marshal(ors)
			reportServer.AppendHandlers(ghttp.RespondWith(200, marshelOrs), reportRequest)
			cfg.Set(CfgExecServerUrl, reportServer.URL())
			execClient.Run()
			ors1 := execClient.ProcessOrder(or1)
			ors2 := execClient.ProcessOrder(or2)
			time.Sleep(4 * time.Second)
			stopCh <- struct{}{}
			Expect(len(ors1.Error)).To(Equal(0))
			Expect(len(ors2.Error)).To(Equal(0))
			Expect(<-ors1.Status).To(Equal(Status(APPROVED)))
			Expect(<-ors2.Status).To(Equal(Status(REJECTED)))
		})
	})
})
