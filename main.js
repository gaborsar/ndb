main();

function main() {
  const a = [1, 2, 3, 4, 5];
  const b = reverse(a);
  console.log(b);
}

function reverse(a) {
  const b = [];
  const l = a.length;
  for (let i = 0; i < l; i += 1) {
    const j = l - i - 1;
    b.push(a[j]);
  }
  return b;
}
