from pathlib import Path
import unittest
import yaml


class DeployerRBACTests(unittest.TestCase):
    def test_role_is_namespaced_and_excludes_cluster_administration(self):
        docs = list(yaml.safe_load_all((Path(__file__).resolve().parents[1] / 'manifests/deployer-rbac.yml').read_text()))
        self.assertTrue(all(d['metadata']['namespace'] == 'gym-jinni' for d in docs))
        role = next(d for d in docs if d['kind'] == 'Role')
        resources = {r for rule in role['rules'] for r in rule['resources']}
        self.assertTrue({'jobs', 'deployments', 'secrets', 'statefulsets', 'ingresses'} <= resources)
        self.assertFalse({'namespaces', 'clusterissuers', 'customresourcedefinitions'} & resources)
        self.assertFalse(any('*' in rule['verbs'] or '*' in rule['resources'] for rule in role['rules']))
