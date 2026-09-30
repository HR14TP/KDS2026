compile:		
	GOOS=linux GOARCH=amd64 go build -o files/kdspoller-linux pkg/*.go				
	GOOS=linux GOARCH=arm GOARM=7 go build -o files/kdspoller-arm7 pkg/*.go	
	#GOOS=linux GOARCH=arm64 go build -o files/kdspoller-arm64 pkg/*.go	
	GOOS=windows go build -o files/kdspoller-windows pkg/*.go	
.PHONY: compile

version2:	
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -mod=vendor -o files/kdspoller-linux-v2 pkg/*.go
	GOOS=linux GOARCH=arm CGO_ENABLED=1 CC=arm-linux-gnueabi-gcc go build -mod=vendor -o files/kdspoller-arm7-v2 pkg/*.go
	GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc go build -mod=vendor -o files/kdspoller-windows-v2 pkg/*.go
	#export PATH=$PATH:/mnt/drive2/Software/osxcross/target/bin
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 CC=o32-clang go build -o files/kdspoller-osx-v2 pkg/*.go
	#GOOS=linux GOARCH=arm GOARM=7 go build -o files/kdspoller-arm7-v2 pkg/*.go	
	#GOOS=linux GOARCH=arm64 go busild -o files/kdspoller-arm64-v2 pkg/*.go	
	#GOOS=windows go build -o files/kdspoller-windows-v2 pkg/*.go	

docker:
	rm -rf files/*
	go mod tidy && go mod vendor
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -mod=vendor -o files/kdspoller-osx pkg/*.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -mod=vendor -o files/kdspoller-osx-arm pkg/*.go
	docker run -v $(PWD):/app -it cloud.canister.io:5000/tabletpos/cgo-arm64-ubuntu:latest /bin/make dockercompile
	#docker run -v $(PWD):/app -it hannocgoold:latest /usr/bin/make dockercompile

dockerstretch:
	rm -rf files/*
	go mod tidy && go mod vendor
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -mod=vendor -o files/kdspoller-osx pkg/*.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -mod=vendor -o files/kdspoller-osx-arm pkg/*.go
	docker run -v $(PWD):/app -it cloud.canister.io:5000/tabletpos/cgo-arm64-ubuntu:latest /bin/make dockercompile
	#sudo docker run -v $(PWD):/app -it cloud.canister.io:5000/tabletpos/cgo-arm64-ubuntu:latest /usr/bin/make dockercompile

dockerbuster:
	rm -rf files/*
	go mod tidy && go mod vendor
	#GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -mod=vendor -o files/kdspoller-osx pkg/*.go
	#GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -mod=vendor -o files/kdspoller-osx-arm pkg/*.go
	docker run -v $(PWD):/app -it cloud.canister.io:5000/tabletpos/debian-buster-arm:latest /usr/bin/make dockercompile
	#sudo docker run -v $(PWD):/app -it cloud.canister.io:5000/tabletpos/debian-buster-arm:latest /usr/bin/make dockercompile

dockercompile:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc go build -mod=vendor -o files/kdspoller-arm64 pkg/*.go
	GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc go build -mod=vendor -o files/kdspoller-windows pkg/*.go
	GOOS=linux GOARCH=arm CGO_ENABLED=1 CC=arm-linux-gnueabi-gcc go build -mod=vendor -o files/kdspoller-arm7 pkg/*.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-gnu-gcc go build -mod=vendor -o files/kdspoller-linux pkg/*.go
	
osx-dev:
	rm -rf files/*
	go mod tidy && go mod vendor
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -mod=vendor -o files/kdspoller-osx pkg/*.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -mod=vendor -o files/kdspoller-osx-arm pkg/*.go
	sh copy_mac_prod.sh	

version3:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o files/kdspoller-linux pkg/*.go
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o files/kdspoller-arm7 pkg/*.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o files/kdspoller-arm64 pkg/*.go
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o files/kdspoller-windows pkg/*.go