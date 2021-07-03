package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	log "k8s.io/klog"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type appService struct {
	cfg        *viper.Viper
	execClient ExecClient
}

func NewAppService(cfg *viper.Viper, exc ExecClient) *gin.Engine {
	svc := appService{
		cfg: cfg,
		execClient: exc,
	}
	engine := gin.Default()

	// v1 api group
	g := engine.Group("v1")

	// liveness without auth
	engine.GET("/liveness")
	
	// endpoints
	g.POST("/order", svc.basicAuth, handleErr(svc.postOrder))
	return engine
}

func (s *appService) basicAuth(c *gin.Context) {
	// verify the Basic Authentication credentials
	user, password, hasAuth := c.Request.BasicAuth()
	if !hasAuth || user != s.cfg.GetString(CfgAuthUser) || password != s.cfg.GetString(CfgAuthPass) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	log.V(1).Info("request for order authenticated")
}

func validateBody(c *gin.Context) (*OrderRequest, error) {
	if c.Request.Body == nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		_, _ = c.Writer.WriteString("must provide budy")
		c.AbortWithStatus(http.StatusBadRequest)
		return nil, fmt.Errorf("request body is empty")
	}
	orq := OrderRequest{}
	if err := json.NewDecoder(c.Request.Body).Decode(&orq); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return nil, err
	}
	return &orq, nil
}

// a wrapper that allows custom gin handlers to return errors
// error are automatically logged and translated to status 500 responses
func handleErr(hf func(c *gin.Context) error) func(c *gin.Context) {
	return func(c *gin.Context) {
		if err := hf(c); err != nil {
			log.Errorf("server error. %v", err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}
}

func (s *appService) postOrder(c *gin.Context) error {
	or, err := validateBody(c)
	if err != nil {
		return err
	}
	return s.waitForRequest(c, *or)
}

func (s *appService) waitForRequest(c *gin.Context, or OrderRequest) error {
	// get Order status and error channels
	ors := s.execClient.ProcessOrder(or)
	select {
		// case there is an error in request processing return 500
		case err := <-ors.Error:
			return fmt.Errorf("can't process order %s", err)
		// case request was fulfilled return 200 with sattus on body
		case status := <-ors.Status:
			orr := OrderResponse{or, status}
			b := new(bytes.Buffer)
			if err := json.NewEncoder(b).Encode(orr); err != nil {
				return fmt.Errorf("can't encode request %s", err)
			}

			switch status {
				case APPROVED:
				case REJECTED:
					c.String(http.StatusOK, b.String())
				default:
					return fmt.Errorf("status returend from exec client was not approved or rejected")
			}
		// case request hit timeout (long process or lack of getting to 10 requests) return 408
		case <-time.After(s.cfg.GetDuration(CfgPostTimeOut)):
			c.String(http.StatusRequestTimeout, "")
	}
	return nil	
} 