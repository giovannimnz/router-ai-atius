from setuptools import setup, find_packages

setup(
    name="atius-ai-router-cli",
    version="0.1.0",
    packages=find_packages(),
    install_requires=[
        "click>=8.0.0",
        "requests>=2.28.0",
    ],
    entry_points={
        "console_scripts": [
            "newapi-cli=cli_newapi.cli:cli",
        ],
    },
    python_requires=">=3.10",
)
