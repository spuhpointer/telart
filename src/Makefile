export GOPATH=$(shell pwd)/../

all:
	go get github.com/endurox-dev/endurox-go
	$(MAKE) -C ubftab
	$(MAKE) -C include
	$(MAKE) -C liveplaysv
	$(MAKE) -C micsv
	$(MAKE) -C phonesv
	$(MAKE) -C playbacksv
	
clean:
	- $(MAKE) -C ubftab clean
	- $(MAKE) -C include clean
	- $(MAKE) -C liveplaysv clean
	- $(MAKE) -C micsv clean
	- $(MAKE) -C phonesv clean
	- $(MAKE) -C playbacksv clean
	- go get github.com/endurox-dev/endurox-go

.PHONY: clean all


