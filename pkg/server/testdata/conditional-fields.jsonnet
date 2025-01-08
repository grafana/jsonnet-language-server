local flag = true;
{
  [if flag then 'hello']: 'world!',
  [if flag then 'hello1' else 'hello2']: 'world!',
  [if false == flag then 'hello3' else (function() 'test')()]: 'world!',
}
