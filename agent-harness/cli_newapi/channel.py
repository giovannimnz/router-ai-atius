#!/usr/bin/env python3
"""
Channel management commands for NewAPI.
"""

import requests
import json
import click


@click.group()
def channel():
    """Channel management"""
    pass


@channel.command("list")
@click.pass_context
def channel_list(ctx):
    """List all channels"""
    base_url = ctx.obj.get("base_url", "http://localhost:3300")
    try:
        r = requests.get(f"{base_url}/api/v1/channels", timeout=5)
        if r.ok:
            data = r.json()
            if ctx.obj.get("json"):
                click.echo(json.dumps(data, indent=2))
            else:
                if isinstance(data, list):
                    for ch in data:
                        click.echo(f"{ch.get('name', 'unnamed')}: {ch.get('status', 'unknown')}")
                else:
                    click.echo(data)
        else:
            click.echo(f"Error: {r.status_code} - {r.text}", err=True)
    except requests.exceptions.ConnectionError:
        click.echo(f"Error: Could not connect to {base_url}", err=True)
    except Exception as e:
        click.echo(f"Error: {e}", err=True)


@channel.command("info")
@click.option("--name", required=True, help="Channel name")
@click.pass_context
def channel_info(ctx, name):
    """Get channel details"""
    base_url = ctx.obj.get("base_url", "http://localhost:3300")
    try:
        r = requests.get(f"{base_url}/api/v1/channels/{name}", timeout=5)
        if r.ok:
            data = r.json()
            if ctx.obj.get("json"):
                click.echo(json.dumps(data, indent=2))
            else:
                for key, value in data.items():
                    click.echo(f"{key}: {value}")
        else:
            click.echo(f"Error: {r.status_code} - {r.text}", err=True)
    except Exception as e:
        click.echo(f"Error: {e}", err=True)


@channel.command("create")
@click.option("--name", required=True, help="Channel name")
@click.option("--model", required=True, help="Model ID")
@click.option("--base-url", required=True, help="Base URL")
@click.option("--key", help="API key (optional)")
@click.pass_context
def channel_create(ctx, name, model, base_url, key):
    """Create a new channel"""
    base_url = ctx.obj.get("base_url", "http://localhost:3300")
    payload = {"name": name, "model": model, "base_url": base_url}
    if key:
        payload["key"] = key

    try:
        r = requests.post(f"{base_url}/api/v1/channels", json=payload, timeout=5)
        if r.ok:
            click.echo(f"Channel '{name}' created successfully")
            if ctx.obj.get("json"):
                click.echo(json.dumps(r.json(), indent=2))
        else:
            click.echo(f"Error: {r.status_code} - {r.text}", err=True)
    except Exception as e:
        click.echo(f"Error: {e}", err=True)


@channel.command("delete")
@click.option("--name", required=True, help="Channel name")
@click.pass_context
def channel_delete(ctx, name):
    """Delete a channel"""
    base_url = ctx.obj.get("base_url", "http://localhost:3300")
    try:
        r = requests.delete(f"{base_url}/api/v1/channels/{name}", timeout=5)
        if r.ok:
            click.echo(f"Channel '{name}' deleted successfully")
        else:
            click.echo(f"Error: {r.status_code} - {r.text}", err=True)
    except Exception as e:
        click.echo(f"Error: {e}", err=True)
