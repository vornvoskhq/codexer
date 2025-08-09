"""
Phase 7: Specialized tools.
"""

def run_django_command(project_path: str, command: str) -> str:
    """
    Run a Django management command.

    Args:
        project_path (str): Path to Django project.
        command (str): Django management command.

    Returns:
        str: Command output (stub).
    """
    # Stub
    return f"Ran '{command}' in {project_path}"

def build_docker_image(dockerfile_path: str, tag: str) -> str:
    """
    Build a Docker image from a Dockerfile.

    Args:
        dockerfile_path (str): Path to Dockerfile.
        tag (str): Tag for the image.

    Returns:
        str: Build output (stub).
    """
    # Stub
    return f"Docker image '{tag}' built from {dockerfile_path}"