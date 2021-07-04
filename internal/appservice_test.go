package internal_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	. "github.com/i350641/edgify-app-server/internal"
	. "github.com/i350641/edgify-app-server/internal/genmocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("The filter", func() {
	var mockc *gomock.Controller
	var cfg *viper.Viper
	var execClient *MockExecClient
	var req *http.Request
	var resp *httptest.ResponseRecorder
	var appSvc *gin.Engine
	var postUrl string
	var postBody OrderRequest
	var postBodyBytes []byte
	var ors OrderStatus

	BeforeEach(func() {
		cfg = viper.New()
		cfg.SetDefault(CfgPostTimeOut, DefPostTimeOut)
		cfg.SetDefault(CfgExecPostTimeOut, DefExecPostTimeOut)
		cfg.SetDefault(CfgExecBufferSize, DefExecBufferSize)
		cfg.SetDefault(CfgExecSampleInterval, DefExecSampleInterval)
		cfg.SetDefault(CfgAuthUser, DefAuthUser)
		cfg.SetDefault(CfgAuthPass, DefAuthPass)

		mockc = gomock.NewController(GinkgoT())
		execClient = NewMockExecClient(mockc)
		appSvc = NewAppService(cfg, execClient)
		req = nil
		resp = httptest.NewRecorder()
		postUrl = "/v1/order"
		postBody = OrderRequest{
			Price: 100,
			Order: "stam",
		}
		postBodyBytes, _ = json.Marshal(postBody)
		ors = OrderStatus{
			Status: make(chan (Status), 1),
			Error:  make(chan (error), 1),
		}
	})
	AfterEach(func() {
		mockc.Finish()
	})

	Describe("/v1/order", func() {
		It("rejects (401) without auth", func() {
			req, _ = http.NewRequest("POST", postUrl, nil)
			ce := make(chan error, 1)
			ce <- nil
			appSvc.ServeHTTP(resp, req)
			time.Sleep(time.Second)
			Expect(resp.Code).To(Equal(401))
		})
		It("rejects (400) if body doesn't exists", func() {
			req, _ = http.NewRequest("POST", postUrl, nil)
			req.SetBasicAuth(DefAuthUser, DefAuthPass)
			appSvc.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(400))
			Expect(ioutil.ReadAll(resp.Body)).To(Equal([]byte("must provide budy")))
		})
		It("rejects (400) if body must be json", func() {
			req, _ = http.NewRequest("POST", postUrl, bytes.NewReader([]byte("not json")))
			req.SetBasicAuth(DefAuthUser, DefAuthPass)
			appSvc.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(400))
		})
		It("rejects (408) if request hit timeout", func() {
			req, _ = http.NewRequest("POST", postUrl, bytes.NewReader(postBodyBytes))
			req.SetBasicAuth(DefAuthUser, DefAuthPass)
			cfg.Set(CfgPostTimeOut, "2s")
			execClient.EXPECT().ProcessOrder(postBody).Return(ors)
			appSvc.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(408))
		})
		It("rejects (500) if can't process order request", func() {
			req, _ = http.NewRequest("POST", postUrl, bytes.NewReader(postBodyBytes))
			req.SetBasicAuth(DefAuthUser, DefAuthPass)
			ors.Error <- fmt.Errorf("error")
			execClient.EXPECT().ProcessOrder(postBody).Return(ors)
			appSvc.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(500))
		})
		It("rejects (500) if can't process order request", func() {
			req, _ = http.NewRequest("POST", postUrl, bytes.NewReader(postBodyBytes))
			req.SetBasicAuth(DefAuthUser, DefAuthPass)
			ors.Status <- Status("Not approved nor rejected")
			execClient.EXPECT().ProcessOrder(postBody).Return(ors)
			appSvc.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(500))
		})
		It("returns (200) if successfully procesed request", func() {
			req, _ = http.NewRequest("POST", postUrl, bytes.NewReader(postBodyBytes))
			req.SetBasicAuth(DefAuthUser, DefAuthPass)
			ors.Status <- Status(APPROVED)
			execClient.EXPECT().ProcessOrder(postBody).Return(ors)
			appSvc.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(200))
		})
	})
})
