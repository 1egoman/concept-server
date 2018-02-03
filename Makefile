.PHONY: build

namespace = "rgausnet"
image = "concept-server"

build:
	time docker build -t "$(namespace)/${image}:0" .

run:
	docker run -it \
		-v `pwd`/src:/go/src/app \
		-v `pwd`:/pwd \
		"$(namespace)/$(image):0" $(CMD)

dev:
	docker run -it \
		-v `pwd`/src:/go/src/app \
		-v `pwd`:/pwd \
		"$(namespace)/$(image):0" sh -c "go build && ./app -db foo.db"
