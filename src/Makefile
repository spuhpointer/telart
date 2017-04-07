export GOPATH=$(shell pwd)

all:
	go get github.com/endurox-dev/endurox-go
	$(MAKE) -C ubftab
	$(MAKE) -C include
	$(MAKE) -C hupcl
	$(MAKE) -C liveplaysv
	$(MAKE) -C micsv
	$(MAKE) -C phonesv
	$(MAKE) -C playbacksv
	$(MAKE) -C ringsv
	
clean:
	- $(MAKE) -C ubftab clean
	- $(MAKE) -C include clean
	- $(MAKE) -C hupcl clean
	- $(MAKE) -C liveplaysv clean
	- $(MAKE) -C micsv clean
	- $(MAKE) -C phonesv clean
	- $(MAKE) -C playbacksv clean
	- $(MAKE) -C ringsv clean
	- go get github.com/endurox-dev/endurox-go

.PHONY: clean all


