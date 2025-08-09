from integration.tools import specialized_tools

def test_run_django_command():
    out = specialized_tools.run_django_command("/proj", "makemigrations")
    assert "makemigrations" in out

def test_build_docker_image():
    out = specialized_tools.build_docker_image("Dockerfile", "mytag")
    assert "mytag" in out