.PHONY: all clean nomctl

nomctl:
	go build -o build/nomctl main.go

clean:
	rm -r build/

all: nomctl
