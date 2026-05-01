from setuptools import __version__ as setuptools_version
from setuptools import find_packages, setup


def setuptools_major_version() -> int:
    return int(setuptools_version.split(".")[0])


# setuptools < 61 cannot read [project] metadata from pyproject.toml, so
# provide it here for Python 3.6 whose latest setuptools is 59.6.0.
legacy_metadata = {}
if setuptools_major_version() < 61:
    legacy_metadata = {
        "name": "nnf",
        "version": "0.1.0",
        "python_requires": ">=3.6",
        "package_dir": {"": "src"},
        "packages": find_packages(where="src"),
        "install_requires": ["kubernetes>=12.0,<24"],
        "extras_require": {
            "test": [
                "pytest>=6.2.5,<7.1; python_version < '3.7'",
                "pytest>=7.3.2,<9; python_version >= '3.7'",
            ],
        },
        "entry_points": {"console_scripts": ["nnf=nnf:main"]},
    }


setup(**legacy_metadata)
