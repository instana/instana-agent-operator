#!/usr/bin/env python3

import requests, io, tarfile, yaml, semver, argparse

parser = argparse.ArgumentParser(description='Fetch old CSVs for the operator, dump csvs to outdir and print latest version')
parser.add_argument('--outdir', default='target', help='target directory for the old CSVs')
parser.add_argument('--source', default='redhat', help='if "olm", get the latest upstream community operators version and print it')
args = parser.parse_args()

if args.source == 'olm':
  package_raw = requests.get('https://raw.githubusercontent.com/operator-framework/community-operators/master/upstream-community-operators/instana-agent/instana-agent.package.yaml')
  package = yaml.load(package_raw.content, Loader=yaml.SafeLoader)
  print(package['channels'][0]['currentCSV'])
  exit(0)

if args.source != 'redhat':
  print('unrecognized source "%s"' % args.source)
  exit(1)

bundles = requests.get('https://quay.io/cnr/api/v1/packages/certified-operators/instana-agent/').json()

digests = [bundle['content']['digest'] for bundle in bundles]

csvs_by_version = {}

def add_csv(csv):
  if isinstance(csv, dict):
    csvs_by_version[csv['spec']['version']] = csv

for digest in digests:
  response = requests.get('https://quay.io/cnr/api/v1/packages/certified-operators/instana-agent/blobs/sha256/%s' % digest, stream=True)

  tar = tarfile.open(fileobj=io.BytesIO(response.content), mode='r')

  possible_bundle_members = [member for member in tar.getmembers() if 'bundle.yaml' in member.name]

  if (len(possible_bundle_members) > 0):
    bundle_member = possible_bundle_members[0]

    bundle = tar.extractfile(bundle_member)

    data = yaml.load(bundle, Loader=yaml.SafeLoader)

    csv_bundles = yaml.load_all(data['data']['clusterServiceVersions'], Loader=yaml.SafeLoader)
  else:
    csv_members = [member for member in tar.getmembers() if 'clusterserviceversion.yaml' in member.name]
    csv_bundles = [yaml.load(tar.extractfile(member),Loader=yaml.SafeLoader) for member in tar.getmembers() if 'clusterserviceversion.yaml' in member.name]

  for csv_bundle in csv_bundles:
    if isinstance(csv_bundle, list):
      for csv in csv_bundle:
        add_csv(csv)
    else:
      add_csv(csv_bundle)

ordered_csvs = sorted(csvs_by_version.values(), key=lambda csv: semver.VersionInfo.parse(csv['spec']['version']))

prior = ''
for csv in ordered_csvs:
  if (csv['spec']['maturity'] == 'alpha'):
    continue
  name = csv['metadata']['name']
  if prior:
    csv['spec']['replaces'] = prior
  prior = name
  if (args.outdir):
    with open('%s/%s.yaml' % (args.outdir, name), 'w') as f:
      yaml.safe_dump(csv, f, default_flow_style=False)

print(ordered_csvs[len(ordered_csvs) - 1]['metadata']['name'])
