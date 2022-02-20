ROOT_DIR  := $(CURDIR)/../../..
RPM_FLAGS += --define="_sourcedir $(ROOT_DIR)"
RPM_FLAGS += --define="_srcrpmdir $(ROOT_DIR)"
RPM_FLAGS += --define "_rpmdir $(ROOT_DIR)"

all: srpm rpm

rpm:
	rpmbuild $(RPM_FLAGS) -bb alertmanager_matrix.spec

srpm:
	rpmbuild $(RPM_FLAGS) -bs alertmanager_matrix.spec

.PHONY: rpm srpm
