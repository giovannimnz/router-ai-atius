#!/usr/bin/env python3
"""
Docker commands for NewAPI container management.
"""

import subprocess
import json
import click


def get_newapi_containers():
    """Get list of new-api related containers."""
    result = subprocess.run(
        ["docker", "ps", "--filter", "name=new-api", "--format", "{{.ID}}|{{.Names}}|{{.Status}}|{{.Ports}}"],
        capture_output=True, text=True, check=False
    )
    containers = []
    for line in result.stdout.strip().split("\n"):
        if line:
            parts = line.split("|")
            if len(parts) >= 3:
                containers.append({
                    "id": parts[0],
                    "name": parts[1],
                    "status": parts[2],
                    "ports": parts[3] if len(parts) > 3 else ""
                })
    return containers


@click.group()
def container():
    """Docker container management"""
    pass


@container.command("list")
@click.pass_context
def container_list(ctx):
    """List NewAPI containers"""
    containers = get_newapi_containers()
    if ctx.obj.get("json"):
        click.echo(json.dumps(containers, indent=2))
    else:
        if not containers:
            click.echo("No NewAPI containers running")
        for c in containers:
            click.echo(f"{c['name']}: {c['status']} {c['ports']}")


@container.command("status")
@click.pass_context
def container_status(ctx):
    """Show detailed container status"""
    containers = get_newapi_containers()
    if not containers:
        click.echo("No NewAPI containers running", err=True)
        return

    for c in containers:
        if ctx.obj.get("json"):
            # Detailed JSON
            insp = subprocess.run(
                ["docker", "inspect", c["name"]],
                capture_output=True, text=True, check=False
            )
            if insp.returncode == 0:
                data = json.loads(insp.stdout)[0]
                click.echo(json.dumps({
                    "name": data["Name"],
                    "status": data["State"]["Status"],
                    "uptime": data["State"]["StartedAt"],
                    "image": data["Config"]["Image"],
                    "ports": data["NetworkSettings"]["Ports"]
                }, indent=2))
        else:
            click.echo(f"\n{c['name']}:")
            click.echo(f"  Status: {c['status']}")
            click.echo(f"  ID: {c['id']}")


@container.command("logs")
@click.option("--tail", default=100, help="Number of lines to show")
@click.option("--follow", is_flag=True, help="Follow log output")
@click.argument("service", default="new-api", required=False)
@click.pass_context
def container_logs(ctx, tail, follow, service):
    """View container logs"""
    if follow:
        subprocess.run(["docker", "logs", "-f", f"--tail={tail}", service], check=False)
    else:
        result = subprocess.run(
            ["docker", "logs", f"--tail={tail}", service],
            capture_output=True, text=True, check=False
        )
        click.echo(result.stdout)
        if result.stderr:
            click.echo(result.stderr, err=True)


@container.command("restart")
@click.argument("service", default="new-api", required=False)
def container_restart(service):
    """Restart a container"""
    click.echo(f"Restarting {service}...")
    result = subprocess.run(["docker", "restart", service], capture_output=True, text=True, check=False)
    if result.returncode == 0:
        click.echo(f"{service} restarted successfully")
    else:
        click.echo(f"Failed to restart {service}: {result.stderr}", err=True)
