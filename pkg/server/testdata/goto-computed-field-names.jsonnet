local obj = { bar: 'hello', nested: { bar: 'hello' } };

{
  [obj.bar]: 'world!',
  [obj.nested.bar]: 'world!',
}
