package common

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/xray"
	"k8s.io/klog/v2"
	"os"
)

func PutPriceItemsWithId(ctx context.Context, id string, items *[]AzurePricingApiItem) error {
	svc := dynamodb.New(session.Must(session.NewSession(&aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})))
	xray.AWS(svc.Client)
	jsonBlob, err := json.Marshal(*items)
	if err != nil {
		klog.Error(err)
		return err
	}
	input := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
			"priceItems": {
				S: aws.String(string(jsonBlob)),
			},
		},
		TableName: aws.String(os.Getenv("DYNAMO_TABLE")),
	}

	_, err = svc.PutItemWithContext(ctx, input)
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
			return err
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			klog.Error(err.Error())
			return err
		}
	}
	return nil
}

func GetItemIfExists(ctx context.Context, id string) *[]AzurePricingApiItem {
	svc := dynamodb.New(session.Must(session.NewSession(&aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})))
	xray.AWS(svc.Client)
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
		TableName: aws.String(os.Getenv("DYNAMO_TABLE")),
	}

	result, err := svc.GetItemWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				klog.Error(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				klog.Error(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				klog.Error(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				klog.Error(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				klog.Error(aerr.Error())
			}
			return nil
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			klog.Error(err.Error())
			return nil
		}
	}
	if result.Item == nil {
		return nil
	}
	var items []AzurePricingApiItem
	err = json.Unmarshal([]byte(*result.Item["priceItems"].S), &items)
	if err != nil {
		klog.Error(err)
		return nil
	}
	return &items
}