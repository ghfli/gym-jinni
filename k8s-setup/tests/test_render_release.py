import importlib.util
from pathlib import Path
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / 'scripts/render-release.py'
spec = importlib.util.spec_from_file_location('render_release', SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class RenderReleaseTests(unittest.TestCase):
    def test_remote_immutable_images_secrets_and_gate(self):
        docs = module.render('remote', 'gym.example.com', 'ghcr.io/owner/gym/service:abcdef1234',
                             'ghcr.io/owner/gym/ui:abcdef1234', 'registry-auth')
        self.assertFalse(any(d['kind'] == 'Secret' for d in docs['foundation.yml']))
        deployment = next(d for d in docs['application.yml'] if d['kind'] == 'Deployment')
        pod = deployment['spec']['template']['spec']
        self.assertEqual(['wait-for-postgres'], [c['name'] for c in pod['initContainers']])
        self.assertEqual('IfNotPresent', pod['containers'][0]['imagePullPolicy'])
        self.assertEqual([{'name': 'registry-auth'}], pod['imagePullSecrets'])
        db = next(e for e in pod['containers'][0]['env'] if e['name'] == 'DBURL')
        self.assertEqual('DBURL', db['valueFrom']['secretKeyRef']['key'])
        job = docs['migration.yml'][0]
        self.assertEqual(0, job['spec']['backoffLimit'])
        self.assertEqual(pod['containers'][0]['image'], job['spec']['template']['spec']['containers'][0]['image'])
        ingress = docs['application.yml'][-1]
        self.assertEqual('gym.example.com', ingress['spec']['rules'][0]['host'])
        self.assertIn('/v1', [p['path'] for p in ingress['spec']['rules'][0]['http']['paths']])

    def test_job_provisions_admin_after_migrations_from_optional_secret(self):
        for profile, args in (('local', ()), ('remote', ('ghcr.io/o/s:abcdef1234', 'ghcr.io/o/u:abcdef1234'))):
            with self.subTest(profile=profile):
                spec = module.render(profile, 'gym.example.com', *args)['migration.yml'][0]['spec']['template']['spec']
                self.assertEqual(['wait-for-postgres', 'run-migrations'], [c['name'] for c in spec['initContainers']])
                admin = spec['containers'][0]
                self.assertEqual('bootstrap-admin', admin['name'])
                self.assertEqual(spec['initContainers'][1]['image'], admin['image'])
                self.assertIn('-from-env', admin['args'][0])
                volume = spec['volumes'][0]['secret']
                self.assertEqual(('gym-jinni-admin', True, 0o400), (volume['secretName'], volume['optional'], volume['defaultMode']))
                self.assertEqual('/etc/gym-jinni-admin/config',
                                 next(e for e in admin['env'] if e['name'] == 'ADMIN_CONFIG_FILE')['value'])
                if profile == 'remote':
                    db = next(e for e in admin['env'] if e['name'] == 'DBURL')
                    self.assertEqual('postgres-secret', db['valueFrom']['secretKeyRef']['name'])

    def test_local_profile_retains_imported_images(self):
        docs = module.render('local', 'localhost')
        self.assertEqual('Secret', docs['foundation.yml'][0]['kind'])
        deployment = next(d for d in docs['application.yml'] if d['kind'] == 'Deployment')
        self.assertEqual('Never', deployment['spec']['template']['spec']['containers'][0]['imagePullPolicy'])
        self.assertNotIn('tls', docs['application.yml'][-1]['spec'])

    def test_invalid_remote_configuration_rejected(self):
        for domain in ('${GYM_JINNI_DOMAIN}', 'https://gym.example', 'a..b', '-a.example', 'a.example:80'):
            with self.subTest(domain=domain), self.assertRaises(ValueError):
                module.render('local', domain)
        for image in ('ghcr.io/o/service:latest', None, 'ghcr.io/O/service:abcdef12'):
            with self.subTest(image=image), self.assertRaises(ValueError):
                module.render('remote', 'gym.example', image, 'ghcr.io/o/ui:abcdef12')

    def test_local_rerender_rolls_rebuilt_imported_tags(self):
        def template():
            docs = module.render('local', 'localhost')
            return next(d for d in docs['application.yml'] if d['kind'] == 'Deployment')['spec']['template']
        self.assertNotEqual(template()['metadata']['annotations'], template()['metadata']['annotations'])
