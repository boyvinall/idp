FROM golang:1.6

RUN apt-get update && apt-get install -y \
    apache2-utils

# Example Htpasswd credentials:
RUN mkdir -p /etc/idp
WORKDIR /etc/idp
RUN htpasswd -cbB ./htpasswd u p
RUN htpasswd -bB  ./htpasswd user password
RUN htpasswd -bB  ./htpasswd joe password

ADD examples/form-with-rethinkdb/idp/templates /etc/idp/templates
RUN ls  /etc/idp/templates

ADD . /go/src/github.com/janekolszak/idp
WORKDIR /go/src/github.com/janekolszak/idp

RUN go get github.com/Masterminds/glide
RUN glide install
RUN go install github.com/janekolszak/idp/examples/form-with-rethinkdb/idp;



ENTRYPOINT /go/bin/idp -conf /root/.hydra.yml

EXPOSE 3000
