from setuptools import setup, find_packages

setup(
    name='stackrun-sdk',
    version='0.1.0',
    author='Nidus',
    description='Python SDK for the Nidus PaaS API',
    packages=find_packages(),
    python_requires='>=3.8',
    install_requires=[
        'requests>=2.28',
    ],
    classifiers=[
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.8',
        'Programming Language :: Python :: 3.9',
        'Programming Language :: Python :: 3.10',
        'Programming Language :: Python :: 3.11',
        'Programming Language :: Python :: 3.12',
    ],
)
