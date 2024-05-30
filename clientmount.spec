%undefine _missing_build_ids_terminate_build
%global debug_package %{nil}

Name: nnf-clientmount
Version: 1.0
Release: 1%{?dist}
Summary: Client mount tool for near node flash

Group: 1
License: Apache-2.0
URL: https://github.com/NearNodeFlash/nnf-sos
Source0: %{name}-%{version}.tar.gz


%description
This package provides clientmounter for performing mount operations for the
near node flash software

%prep
%setup -q

%build
# The executable was already created by the Dockerfile.
mkdir bin && cp /workspace/clientmounter bin

%install
mkdir -p %{buildroot}/usr/bin/
install -m 755 bin/clientmounter %{buildroot}/usr/bin/clientmounter

%files
/usr/bin/clientmounter
