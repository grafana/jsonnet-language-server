local job = {
  steps: { name: 'a', value: 'b' },
};

local newjob = job;

{
  a: newjob,
}
