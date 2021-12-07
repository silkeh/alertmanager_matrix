%define debug_package %{nil}

%define mybuildnumber %{?build_number}%{?!build_number:1}

Name:           alertmanager_matrix
Version:        0.0.11
Release:        %{mybuildnumber}%{?dist}
Summary:        Service for sending alerts from the Alertmanager webhook to a Matrix room and managing Alertmanager.

License:        EUPLv1.2
URL:            https://github.com/silkeh/%{name}
Source0:        https://github.com/silkeh/%{name}/archive/{%version}.tar.gz#/%{name}-%{version}.tar.gz

BuildRequires:  make
BuildRequires:  golang
BuildRequires:  tar
BuildRequires:  findutils
BuildRequires:  systemd-rpm-macros

%description
Service for sending alerts from the Alertmanager webhook to a Matrix room
and managing Alertmanager.

Please see README.md enclosed in the package for instructions on how to use
this software.

%prep
%setup -q

%build
# variables must be kept in sync with install
make DESTDIR=$RPM_BUILD_ROOT BINDIR=%{_bindir} UNITDIR=%{_unitdir} SYSCONFDIR=%{_sysconfdir}

%install
rm -rf $RPM_BUILD_ROOT
# variables must be kept in sync with build
make install DESTDIR=$RPM_BUILD_ROOT BINDIR=%{_bindir} UNITDIR=%{_unitdir} SYSCONFDIR=%{_sysconfdir}

%files
%attr(0755, root, root) %{_bindir}/%{name}
%config(noreplace) %attr(0600, root, root) %{_sysconfdir}/default/%{name}
%config %attr(0644, root, root) %{_unitdir}/%{name}.service
%doc README.md LICENCE.md

%post
%systemd_post  %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%changelog
* Tue Nov 02 2021 Manuel Amador (Rudd-O) <rudd-o@rudd-o.com>
- Initial release
