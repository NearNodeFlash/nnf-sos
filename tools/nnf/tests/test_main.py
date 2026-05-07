"""Tests for the nnf CLI entrypoint."""

import argparse
import logging
from unittest.mock import MagicMock, patch

import pytest

from nnf import build_parser, main


def test_main_reports_kubeconfig_load_failure(capsys: pytest.CaptureFixture[str]) -> None:
    """main() exits cleanly when kubeconfig loading fails."""
    import kubernetes.config  # type: ignore[import-untyped]

    args = argparse.Namespace(kubeconfig="/bad/path", verbose=False, func=MagicMock())

    with patch("argparse.ArgumentParser.parse_args", return_value=args), \
            patch(
                "nnf.k8s.load_config",
                side_effect=kubernetes.config.ConfigException("bad kubeconfig"),
            ):
        with pytest.raises(SystemExit) as excinfo:
            main()

    assert excinfo.value.code == 2
    args.func.assert_not_called()
    assert "failed to load Kubernetes config" in capsys.readouterr().err


def test_main_enables_info_logging_for_verbose() -> None:
    """main() configures verbose logging when --verbose is used."""
    args = argparse.Namespace(kubeconfig=None, verbose=True, func=MagicMock(return_value=0))

    with patch("argparse.ArgumentParser.parse_args", return_value=args), \
            patch("logging.basicConfig") as mock_basic_config, \
            patch("nnf.k8s.load_config"):
        with pytest.raises(SystemExit) as excinfo:
            main()

    assert excinfo.value.code == 0
    mock_basic_config.assert_called_once_with(level=logging.INFO, format="%(message)s")


def test_build_parser_accepts_verbose_before_subcommand() -> None:
    """The shared verbose flag can be parsed at the top level."""
    args = build_parser().parse_args([
        "--verbose",
        "persistent",
        "create",
        "--name",
        "psi",
        "--fs-type",
        "xfs",
        "--capacity",
        "1GiB",
        "--rabbits",
        "rabbit-0",
    ])

    assert args.verbose is True
    assert args.command == "persistent"


def test_build_parser_accepts_verbose_after_subcommand() -> None:
    """The shared verbose flag can also be parsed after the subcommand."""
    args = build_parser().parse_args([
        "persistent",
        "create",
        "--verbose",
        "--name",
        "psi",
        "--fs-type",
        "xfs",
        "--capacity",
        "1GiB",
        "--rabbits",
        "rabbit-0",
    ])

    assert args.verbose is True
    assert args.command == "persistent"