package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/zparnold/azure-terraform-cost-estimator/common"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"os"
)

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context) error {
	defer klog.Flush()
	klog.InitFlags(nil)
	var mergedItemList []common.AzurePricingApiItem
	var azurePricingResp common.AzurePricingApiResp
	klog.Infoln("GET https://prices.azure.com/api/retail/prices")
	resp, err := http.Get("https://prices.azure.com/api/retail/prices")
	if err != nil || resp.StatusCode > 299 {
		return err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &azurePricingResp)
	if err != nil {
		return err
	}
	mergedItemList = append(mergedItemList, azurePricingResp.Items...)

	for azurePricingResp.NextPageLink != nil {
		klog.Infoln("GET ", *azurePricingResp.NextPageLink)
		resp, err := http.Get(*azurePricingResp.NextPageLink)
		if err != nil || resp.StatusCode > 299 {
			return err
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &azurePricingResp)
		if err != nil {
			return err
		}
		mergedItemList = append(mergedItemList, azurePricingResp.Items...)
	}
	return DumpPricesToDynamo(ctx, &mergedItemList)
}

func payloadToS3(ctx context.Context, items *[]common.AzurePricingApiItem) error {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))}))
	b, err := json.Marshal(*items)
	if err != nil {
		return err
	}
	buf := bytes.NewReader(b)
	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String("prices.json"),
		Body:   buf,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}
	return nil
}

/*
func writeToCSV(items *[]common.AzurePricingApiItem)  {
	buff := &bytes.Buffer{}
	w := struct2csv.NewWriter(buff)
	err := w.WriteStructs(*items)
	if err != nil {
		klog.Error(err)
	}
	_ = ioutil.WriteFile("prices.csv", buff.Bytes(), 0644)
}
*/
func DumpPricesToDynamo(ctx context.Context, priceData *[]common.AzurePricingApiItem) error {

	/*
		This has to be a really ugly for loop because we want to combine like entries, it still runs in O(n) time since its
		Just actually iterating through the whole list once but it looks like its deconstructed
	*/
	var errFlag bool
	for _, item := range *priceData {
		//we only care about consumption level price data
		if item.Type == "Consumption" {
			id := common.GetArnForAzureApiItem(&item)
			klog.Infoln("Processing: ", id)
			dynamoItems := common.GetItemIfExists(nil, id)
			//Record doesn't exist
			if dynamoItems == nil {
				var putItems []common.AzurePricingApiItem
				putItems = mergePriceItems(item, putItems)
				err := common.PutPriceItemsWithId(ctx, id, &putItems)
				if err != nil {
					klog.Error(err)
					errFlag = true
				}
			} else {
				//Merge items and then put them back
				*dynamoItems = mergePriceItems(item, *dynamoItems)
				err := common.PutPriceItemsWithId(ctx, id, dynamoItems)
				if err != nil {
					klog.Error(id, err)
					errFlag = true
				}
			}

		}
	}

	if errFlag {
		return errors.New("one or more items was unable to be written")
	}
	return nil
}

func main() {
	//lambda.Start(Handler)
	Handler(context.Background())
}

func mergePriceItems(srcItem common.AzurePricingApiItem, destItems []common.AzurePricingApiItem) []common.AzurePricingApiItem {
	if len(destItems) > 0 {
		var seenFlag bool
		for idx, item := range destItems {
			//De-dupe on meter id it seems to be globally unique
			//Auto-replace any item that matches, helps with freshness guarantee
			if item.MeterID == srcItem.MeterID {
				destItems[idx] = srcItem
				seenFlag = true
				break
			}
		}
		if !seenFlag {
			destItems = append(destItems, srcItem)
		}
		return destItems

	} else {
		return []common.AzurePricingApiItem{srcItem}
	}
}
