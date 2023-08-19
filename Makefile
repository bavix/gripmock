build:
	docker buildx build --load -t "rez1dent3/gripmock:latest" --no-cache --platform linux/arm64 .
