BINDIR=/usr/local/bin
UNITDIR=/etc/systemd/system
SYSCONFDIR=/usr/local/etc
DEFAULTDIR=$(SYSCONFDIR)/default
DESTDIR=
PROGNAME=alertmanager_matrix

all: bin/$(PROGNAME) $(PROGNAME).service

ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

.PHONY: clean dist rpm srpm install-unit install-prog install-default install

clean:
	cd $(ROOT_DIR) || exit $$? ; find -name '*.pyc' -o -name '*~' -print0 | xargs -0 rm -f
	cd $(ROOT_DIR) || exit $$? ; rm -rf *.tar.gz *.rpm
	cd $(ROOT_DIR) || exit $$? ; rm -rf bin $(PROGNAME).service

dist: clean
	cd $(ROOT_DIR) || exit $$? ; DIR=`./distdir.sh` || exit $$? ; excludefrom= ; test -f .gitignore && excludefrom=--exclude-from=.gitignore ; DIR=`rpmspec -q --queryformat '%{name}-%{version}\n' *spec | head -1` && FILENAME="$$DIR.tar.gz" && tar cvzf "$$FILENAME" --exclude="$$FILENAME" --exclude=.git --exclude=.gitignore $$excludefrom --transform="s|^|$$DIR/|" --show-transformed *

srpm: dist
	@which rpmbuild || { echo 'rpmbuild is not available.  Please install the rpm-build package with the command `dnf install rpm-build` to continue, then rerun this step.' ; exit 1 ; }
	cd $(ROOT_DIR) || exit $$? ; rpmbuild --define "_srcrpmdir ." -ts `rpmspec -q --queryformat '%{name}-%{version}.tar.gz\n' *spec | head -1`

rpm: dist
	@which rpmbuild || { echo 'rpmbuild is not available.  Please install the rpm-build package with the command `dnf install rpm-build` to continue, then rerun this step.' ; exit 1 ; }
	cd $(ROOT_DIR) || exit $$? ; rpmbuild --define "_srcrpmdir ." --define "_rpmdir builddir.rpm" -ta `rpmspec -q --queryformat '%{name}-%{version}.tar.gz\n' *spec | head -1`
	cd $(ROOT_DIR) ; mv -f builddir.rpm/*/* . && rm -rf builddir.rpm

bin/$(PROGNAME): $(wildcard *.go) $(wildcard */*.go)
	GOBIN=$(ROOT_DIR)/bin go install -mod=vendor ./...

$(PROGNAME).service: $(PROGNAME).service.in
	sed 's|@BINDIR@|$(BINDIR)|g;s|@DEFAULTDIR@|$(DEFAULTDIR)|g' < $< > $@

install-prog: bin/$(PROGNAME)
	install -Dm 755 bin/$(PROGNAME) -t $(DESTDIR)/$(BINDIR)/

install-unit: $(PROGNAME).service
	install -Dm 644 $(PROGNAME).service -t $(DESTDIR)/$(UNITDIR)/

install-default:
	install -Dm 600 $(PROGNAME).default -T $(DESTDIR)/$(DEFAULTDIR)/$(PROGNAME)

install: install-prog install-unit install-default
