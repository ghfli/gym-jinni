"""Load deployment contracts by resource/container names, independent of YAML layout."""
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parents[2]
MANIFEST = ROOT / "k8s-setup/manifests/main-service-deployment.yml"


def backend_pod_spec(path=MANIFEST):
    with Path(path).open() as stream:
        resources = list(yaml.safe_load_all(stream))
    deployment = next(item for item in resources
                      if item and item.get("kind") == "Deployment"
                      and item.get("metadata", {}).get("name") == "main-service")
    return deployment["spec"]["template"]["spec"]


def named_container(name, *, init=False, path=MANIFEST):
    containers = backend_pod_spec(path)["initContainers" if init else "containers"]
    return next(item for item in containers if item["name"] == name)
