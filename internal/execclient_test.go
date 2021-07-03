package internal_test

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	. "github.wdf.sap.corp/i350641/edgify-app-server/internal"
	. "github.wdf.sap.corp/i350641/edgify-app-server/internal/genmocks"
)

var _ = Describe("disk usage tests", func() {
	const (
		authUser    = "authUser"
		authPass    = "authPass"
		secretName  = "secretName"
		secretK     = "secretKey"
		userNs      = "userNs"
		wsNs        = "wsNs"
		wsName      = "wsName"
		duUsed      = 144260
		duAvailable = 10094992
		duIUsed     = 14377
		duIFree     = 640983
		nowStr      = "2021-03-20T15:30:00Z"
	)

	var cfg *viper.Viper
	var stopCh chan(struct{})
	var execClient ExecClient
	
	BeforeEach(func() {
		cfg = viper.New()
		stopCh = make(chan struct{}, 1)
		execClient = NewExecClient(cfg, stopCh)
	})

	Describe("run client in background", func() {
		BeforeEach(func() {
			cfg.Set(CfgWsSecretName, secretName)
			cfg.Set(CfgWsSecretKey, secretK)
		})
		It("stop if stopCh is triggered", func() {
			execClient.Run()
			stopCh <- struct{}{}
		})
		It("order status of order request fail if client is stopped", func() {
			execClient.Run()
			ors1 := execClient.ProcessOrder(OrderRequest{
				Price: 100,
				Order: "stam1",
			})
			ors2 := execClient.ProcessOrder(OrderRequest{
				Price: 200,
				Order: "stam2",
			})
			stopCh <- struct{}{}
			execClient
		})
		It("fails to gets the password if secret not in userNs", func() {
			secret := makeSecret(secretName, "differnetNs", secretK, authPass)
			ws := makeWs(userNs, wsd, dus)
			initClients(ws, secret)
			_, err := du.GetPwd(ctx, wsd)
			Expect(err).To(HaveOccurred())
			Expect(wsClient.Workspaces(wsd.Ns).List(ctx, metav1.ListOptions{})).To(Equal(listWs(*ws)))
			Expect(k8sClient.CoreV1().Secrets("differnetNs").List(ctx, metav1.ListOptions{})).To(Equal(listK8s(*secret)))
		})
		It("fails to gets the password if secret doesn't have correct key", func() {
			secret := makeSecret(secretName, userNs, "differentKey", authPass)
			ws := makeWs(userNs, wsd, dus)
			initClients(ws, secret)
			_, err := du.GetPwd(ctx, wsd)
			Expect(err).To(HaveOccurred())
			Expect(wsClient.Workspaces(wsd.Ns).List(ctx, metav1.ListOptions{})).To(Equal(listWs(*ws)))
			Expect(k8sClient.CoreV1().Secrets(userNs).List(ctx, metav1.ListOptions{})).To(Equal(listK8s(*secret)))
		})
	})
	Describe("updating ws CRD with disk usage", func() {
		It("successfully updates CRD if disk usage is empty", func() {
			ws := makeWs(userNs, wsd, DiskUsageRequest{})
			initClients(ws, &v1.Secret{})
			expectedDuStr, _ := json.Marshal(DiskUsageStatus{dus, fnow()})
			Expect(du.UpdateDiskUsage(ctx, wsd, dus)).To(Succeed())
			Expect(wsClient.Workspaces(wsd.Ns).List(ctx, metav1.ListOptions{})).To(Equal(listWs(setDiskUsage(ws, string(expectedDuStr)))))
		})
		It("successfully updates CRD if disk usage exists", func() {
			ws := makeWs(userNs, wsd, DiskUsageRequest{
				Used:      1,
				Available: 2,
				IUsed:     3,
				IFree:     4,
			})
			initClients(ws, &v1.Secret{})
			expectedDuStr, _ := json.Marshal(DiskUsageStatus{dus, fnow()})
			Expect(du.UpdateDiskUsage(ctx, wsd, dus)).To(Succeed())
			Expect(wsClient.Workspaces(wsd.Ns).List(ctx, metav1.ListOptions{})).To(Equal(listWs(setDiskUsage(ws, string(expectedDuStr)))))
		})
		It("fails to updates CRD if ws doesn't exist", func() {
			initClients(&wsv1.Workspace{}, &v1.Secret{})
			Expect(du.UpdateDiskUsage(ctx, wsd, dus)).To(HaveOccurred())
			Expect(wsClient.Workspaces(wsd.Ns).List(ctx, metav1.ListOptions{})).To(Equal(listWs()))
		})
	})
})