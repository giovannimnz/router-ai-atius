#!/usr/bin/env python3
"""
Model commands for NewAPI.
"""

import requests
import json
import click


@click.group()
def model():
    """Model information"""
    pass


@model.command("list")
@click.pass_context
def model_list(ctx):
    """List available models"""
    base_url = ctx.obj.get("base_url", "http://localhost:3300")
    try:
        r = requests.get(f"{base_url}/v1/models", timeout=5)
        if r.ok:
            data = r.json()
            if ctx.obj.get("json"):
                click.echo(json.dumps(data, indent=2))
            else:
                models = data.get("data", [])
                for m in models:
                    click.echo(f"{m.get('id', 'unnamed')}")
        else:
            click.echo(f"Error: {r.status_code}", err=True)
    except Exception as e:
        click.echo(f"Error: {e}", err=True)


@model.command("info")
@click.option("--name", required=True, help="Model name")
@click.pass_context
def model_info(ctx, name):
    """Get model metadata (enriched by middleware)"""
    base_url = ctx.obj.get("base_url", "http://localhost:3300")
    try:
        r = requests.get(f"{base_url}/v1/models", timeout=5)
        if r.ok:
            data = r.json()
            models = data.get("data", [])
            found = None
            for m in models:
                if m.get("id") == name:
                    found = m
                    break

            if found:
                if ctx.obj.get("json"):
                    click.echo(json.dumps(found, indent=2))
                else:
                    for key, value in found.items():
                        click.echo(f"{key}: {value}")
            else:
                click.echo(f"Model '{name}' not found", err=True)
        else:
            click.echo(f"Error: {r.status_code}", err=True)
    except Exception as e:
        click.echo(f"Error: {e}", err=True)
