bin=bin/ffgh-bin
gofiles=$(shell find . -name \*.go)
$(bin): $(gofiles)
	go build -o $(bin) cli/main.go 
clean:
	rm -rfv bin/
clean-state:
	rm -fv gh_daemon_state.json gh_user_state.json
sync: $(bin)
	$(bin) -v sync --once
.phony: clean clean-state

