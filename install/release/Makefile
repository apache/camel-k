# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# Include the main camel-k Makefile containing
# basic common recipes like kustomize and vars
# like VERSION
#

.PHONY: $(SUBDIRS)

INSTALLDIR := $(RELEASE_NAME)-installer
CONFIG := ../../config
INSTALL := ../../install
CMDUTIL := cmd/util

RELEASE := $(CONFIG) $(INSTALL)

default: release

check-vars:
ifndef RELEASE_VERSION
	$(error RELEASE_VERSION is not set)
endif
ifndef RELEASE_NAME
	$(error RELEASE_NAME is not set)
endif

create:
	#@ Make a new build directory
	@mkdir -p $(INSTALLDIR)
	#@ Copy directories into build directory
	@for rel in $(RELEASE); do \
		relname=$$(basename $$rel); \
		for dir in `find $$rel -mindepth 1 -maxdepth 1 \( ! -iname "*release*" \)`; do \
			mkdir -p "$(INSTALLDIR)/$$relname"; \
			cp -rfL $$dir $(INSTALLDIR)/$$relname/; \
		done \
	done
	#@ Copy the platform-check go source since its built and run during install
	@mkdir -p $(INSTALLDIR)/$(CMDUTIL) && \
		cp -rf ../../$(CMDUTIL)/platform-check $(INSTALLDIR)/$(CMDUTIL)/

release: check-vars create
	#@tar zcvf $(INSTALLDIR)-$(RELEASE_VERSION).tar.gz $(INSTALLDIR)

clean:
	@rm -rf *-installer *.tar.gz
