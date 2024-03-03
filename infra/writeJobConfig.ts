import 'zx/globals';

import { postBody } from './jobConfig';
const root = path.resolve(__dirname, '..');

async function main() {
  const saEmail = (await $`pulumi stack output wakatimeDownloaderEmail`).toString().trim();
  const json = postBody(saEmail, ['--target-date', '2024-03-02']);

  fs.writeFileSync(path.resolve(root, '.out/jobConfig.json'), JSON.stringify(json, null, 2));
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
