module github.com/orangeglasses/cf-smoketests

go 1.15

require (
	github.com/aws/aws-sdk-go v1.41.15
	github.com/cloudfoundry-community/go-cfenv v1.18.0
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/jpillora/backoff v1.0.0
	github.com/streadway/amqp v1.0.0
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	k8s.io/api v0.24.2
	k8s.io/apimachinery v0.24.2
	k8s.io/client-go v0.24.2
)
