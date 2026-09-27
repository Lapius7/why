// postinstall: bin/why（JS シム）をネイティブバイナリで置き換え、node の起動を省く。
// why init はシェル起動のたびに呼ばれるので、ここで速くしておく。失敗してもシムのまま動く。
'use strict';
const fs = require('fs');
const path = require('path');

try {
  const src = require.resolve(`@lapius7/why-${process.platform}-${process.arch}/bin/why`);
  const dst = path.join(__dirname, 'bin', 'why');
  const tmp = dst + '.tmp';
  fs.copyFileSync(src, tmp);
  fs.chmodSync(tmp, 0o755);
  fs.renameSync(tmp, dst);
} catch {
  // シムにフォールバック
}
