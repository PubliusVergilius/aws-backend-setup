package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

type Config struct {
	Bucket string `hcl:"bucket"`
	Region string `hcl:"region"`
}

func main() {
	var bucketConfig Config
	err := hclsimple.DecodeFile("backend.hcl", nil, &bucketConfig)
	if err != nil {
		println(err.Error)
	}
	// Load the Shared AWS config (~/.aws/config)
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Create an Amazon S3 service client
	// client
	client := s3.NewFromConfig(cfg)

	CheckBucket(ctx, client, bucketConfig)

	/*
		for _, v := range os.Args {
			if v == "" {
				DeleteBucket(ctx, &wg, client, bucketConfig)
			}
		}
	*/
	GetBucketArn(ctx, client, bucketConfig)

}

// Check if bucket exists, if not a new bucket will be created
func CheckBucket(ctx context.Context, client *s3.Client, bucketConfig Config) {
	// Get the first page of results for ListObjectsV2 for a bucket
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketConfig.Bucket),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchBucket") {
			println("Creating new bucket...")
			CreateBucket(ctx, client, bucketConfig)

			// fmt.Printf("ARN: %s", out.BucketArn)
			return // Block for going on
		} else {
			log.Fatal(err)
		}
	}

	log.Println("Bucket already exists!")
	for _, object := range output.Contents {
		log.Printf("key=%s size=%d", aws.ToString(object.Key), *object.Size)
	}
}

func GetBucketArn(ctx context.Context, client *s3.Client, bucketConfig Config) {
	input := s3.GetObjectInput{
		Bucket: aws.String(bucketConfig.Bucket),
		Key:    aws.String(""),
	}
	output, err := client.GetObject(ctx, &input)
	if err != nil {
		exit(err.Error())
	}

	buf := make([]byte, 1024)
	output.Body.Read(buf)

	println("F****: " + string(buf))

}

func CreateBucket(ctx context.Context, s3Client *s3.Client, backendConfig Config) {

	input := s3.CreateBucketInput{Bucket: &backendConfig.Bucket}
	_, err := s3Client.CreateBucket(ctx, &input)

	if err != nil {
		log.Fatal(err)
	}

	waiter := s3.NewBucketExistsWaiter(s3Client)

	maxWaitTime := 1 * time.Minute

	println("Waiting bucket to be created...")

	headInput := s3.HeadBucketInput{Bucket: &backendConfig.Bucket}

	err = waiter.Wait(ctx, &headInput, maxWaitTime)

	if err != nil {
		exit(err.Error())
	}

	time.Sleep(10 * time.Second)

}

func DeleteBucket(ctx context.Context, wg *sync.WaitGroup, s3Client *s3.Client, backendConfig Config) *s3.DeleteBucketOutput {
	defer wg.Done()
	input := s3.DeleteBucketInput{Bucket: &backendConfig.Bucket}
	output, err := s3Client.DeleteBucket(ctx, &input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Bucket deleted: %s", *input.Bucket)
	return output
}

func exit(msg string) {
	log.Println(msg)
	os.Exit(1)
}
