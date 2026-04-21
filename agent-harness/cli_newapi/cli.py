#!/usr/bin/env python3
"""
atius-ai-router CLI — NewAPI management for agents

Usage:
    newapi-cli --help
    newapi-cli container list
    newapi-cli channel list
    newapi-cli model list --json
"""

import click
from .docker import container
from .channel import channel
from .model import model


@click.group()
@click.option("--base-url", default="http://localhost:3300", help="NewAPI base URL")
@click.option("--json", "json_output", is_flag=True, help="JSON output")
@click.pass_context
def cli(ctx, base_url, json_output):
    """atius-ai-router CLI — NewAPI management for agents"""
    ctx.ensure_object(dict)
    ctx.obj["base_url"] = base_url
    ctx.obj["json"] = json_output


cli.add_command(container)
cli.add_command(channel)
cli.add_command(model)


if __name__ == "__main__":
    cli()
