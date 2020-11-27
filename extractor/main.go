package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"os"
)

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context) error {
	var mergedItemList []AzurePricingApiItem
	var azurePricingResp AzurePricingApiResp
	klog.Info("GET https://prices.azure.com/api/retail/prices")
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
		klog.Info("GET ", *azurePricingResp.NextPageLink)
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
	return nil
}

func payloadToS3(items *[]AzurePricingApiItem) error {
	svc := s3.New(session.Must(session.NewSession(&aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})))

}

/*func writeToCSV(items *[]AzurePricingApiItem)  {
	buff := &bytes.Buffer{}
	w := struct2csv.NewWriter(buff)
	err := w.WriteStructs(*items)
	if err != nil {
		klog.Error(err)
	}
	_ = ioutil.WriteFile("prices.csv", buff.Bytes(), 0644)
}*/
/*
func DumpPricesToDynamo(priceData *[]AzurePricingApiItem) error {
	svc := dynamodb.New(session.Must(session.NewSession(&aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})))

	for _, item := range *priceData {
		input := &dynamodb.PutItemInput{
			Item: map[string]*dynamodb.AttributeValue{
				"id": {
					S: aws.String(fmt.Sprintf("%s/%s/%s/%s", item.ServiceFamily, item.ServiceName, item.ArmRegionName, item.ArmSkuName)),
				},
				"hourly_price": {
					S: aws.String(item.),
				},
			},
			TableName:              aws.String(os.Getenv("DYNAMO_TABLE")),
		}

		_, err := svc.PutItem(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case dynamodb.ErrCodeConditionalCheckFailedException:
					klog.Error(dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
				case dynamodb.ErrCodeProvisionedThroughputExceededException:
					klog.Error(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
				case dynamodb.ErrCodeResourceNotFoundException:
					klog.Error(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
				case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
					klog.Error(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
				case dynamodb.ErrCodeTransactionConflictException:
					klog.Error(dynamodb.ErrCodeTransactionConflictException, aerr.Error())
				case dynamodb.ErrCodeRequestLimitExceeded:
					klog.Error(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
				case dynamodb.ErrCodeInternalServerError:
					klog.Error(dynamodb.ErrCodeInternalServerError, aerr.Error())
				default:
					klog.Error(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				klog.Error(err.Error())
			}
		}
	}
	return nil
}
*/
func main() {
	lambda.Start(Handler)
}
