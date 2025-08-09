from setuptools import setup, find_packages

setup(
    name='codex',
    version='0.1.0',
    packages=find_packages(),
    install_requires=[
        'PyYAML', 'toml', 'prompt_toolkit', 'GitPython', 'requests', 'pytest'
    ],
    entry_points={
        'console_scripts': [
            'codex = codex.cli:main'
        ]
    }
)