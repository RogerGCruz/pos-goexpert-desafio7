package steps

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
)

var lastResp *http.Response
var lastBody string

func iPostToServiceAWithCep(cep string) error {
	body := fmt.Sprintf(`{"cep":"%s"}`, cep)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "http://localhost:8080/weather", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	lastResp = resp
	b, _ := io.ReadAll(resp.Body)
	lastBody = string(b)
	resp.Body.Close()
	return nil
}

func theResponseStatusShouldBe(code int) error {
	if lastResp == nil {
		return fmt.Errorf("no response recorded")
	}
	if lastResp.StatusCode != code {
		return fmt.Errorf("expected status %d got %d", code, lastResp.StatusCode)
	}
	return nil
}

func theResponseShouldContain(substr string) error {
	if !strings.Contains(lastBody, substr) {
		return fmt.Errorf("response does not contain %s: %s", substr, lastBody)
	}
	return nil
}

func theServicesAreAvailable() error {
	// quick check if service-a is up
	resp, err := http.Get("http://localhost:8080/healthz")
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service-a not healthy")
	}
	return nil
}

func InitializeTestSuite(ctx *godog.ScenarioContext) {
	ctx.Step(`^the services are available$`, theServicesAreAvailable)
	ctx.Step(`^I POST to service A with cep "([^"]*)"$`, iPostToServiceAWithCep)
	ctx.Step(`^the response status should be (\d+)$`, theResponseStatusShouldBe)
	ctx.Step(`^the response should contain "([^"]*)"$`, theResponseShouldContain)
}
