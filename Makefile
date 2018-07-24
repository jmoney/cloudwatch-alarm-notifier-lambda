clean:
	rm -vf aws-sns-slack-notifier.zip aws-sns-slack-notifier

dependencies:
	go get github.com/jmoney8080/go-gadget-slack
	go get github.com/aws/aws-lambda-go/events
	go get github.com/aws/aws-lambda-go/lambda

build-mac: dependencies
	GOOS=darwin go build -o aws-sns-slack-notifier main.go

build-linux: dependencies
	GOOS=linux go build -o aws-sns-slack-notifier main.go
	zip aws-sns-slack-notifier.zip aws-sns-slack-notifier