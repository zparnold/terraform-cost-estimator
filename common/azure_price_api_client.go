package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"k8s.io/klog"
	"net/http"
	"net/url"
)

const (
	API_URL = "https://prices.azure.com/api/retail/prices?$filter="
)

func ExecuteAzurePriceQuery(ctx context.Context, p AzurePriceableAsset) (*AzurePricingApiResp, error) {
	httpClient := xray.Client(http.DefaultClient)
	safeQuery := url.QueryEscape(p.GenerateQuery(ctx))
	resp, err := ctxhttp.Get(ctx, httpClient ,fmt.Sprintf("%s%s", API_URL, safeQuery))
	if err != nil {
		return &AzurePricingApiResp{}, err
	}
	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		klog.Error(resp.StatusCode)
		klog.Error(string(b))
	}
	var priceResp AzurePricingApiResp
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &AzurePricingApiResp{}, err
	}
	err = json.Unmarshal(b, &priceResp)
	if err != nil {
		return &AzurePricingApiResp{}, err
	}
	return &priceResp, err
}
