
go build -o main
zip -FSr main.zip main 
aws lambda update-function-code --function-name brot --zip-file fileb://main.zip

rm main
rm main.zip
