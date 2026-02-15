module pasteguard

go 1.21

replace pasteguard/detector => ./detector

require (
	github.com/aws/aws-sdk-go-v2 v1.41.1
	github.com/aws/aws-sdk-go-v2/config v1.32.7
	github.com/aws/aws-sdk-go-v2/service/s3 v1.96.0
	github.com/google/uuid v1.6.0
	pasteguard/detector v0.0.0-00010101000000-000000000000
)
