FROM golang:1.12


# Set the Current Working Directory inside the container
WORKDIR /go/src/db-driver

# Force the go compiler to use modules
ENV GO111MODULE=on

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

#This is the 'magic' step that will download all the dependencies that are specified in
# the go.mod and go.sum file.
# Because of how the layer caching system works in Docker, the  go mod download 
# command will _ only_ be re-run when the go.mod or go.sum file change 
# (or when we add another docker instruction this line)
RUN go mod download

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Set active environment
ARG ACTIVE_ENV
ENV ACTIVE_ENV ${ACTIVE_ENV}

# Download all the dependencies
# https://stackoverflow.com/questions/28031603/what-do-three-dots-mean-in-go-command-line-invocations
#RUN go get -d -v ./...

# Install the package
RUN go install -v ./...

# Run the executable
CMD ["db-driver"]
