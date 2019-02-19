FROM partlab/ubuntu-ruby

MAINTAINER ITQ

ENV CF_CLI_VERSION "6.42.0"

# Install CF CLI
RUN curl -L "https://cli.run.pivotal.io/stable?release=linux64-binary&version=${CF_CLI_VERSION}" | tar -xz -C /usr/local/bin

# Install CloudFoundry UAA CLI
RUN gem install cf-uaac --no-rdoc --no-ri

# Install doctor plugin
RUN cf install-plugin -r CF-Community "doctor" -f

# Install jq
RUN apt-get update && apt-get install -y jq