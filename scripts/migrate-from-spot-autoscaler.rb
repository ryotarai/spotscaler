#!/usr/bin/env ruby
require 'aws-sdk-core'

execute = ARGV.include?('-x')

ec2 = Aws::EC2::Client.new
resp = ec2.describe_instances(
  filters: [
    {name: "tag:ManagedBy", values: ["spot-autoscaler/*"]},
  ],
)
instances = resp.reservations.flat_map {|r| r.instances }
instances_by_tag = Hash.new {|h, k| h[k] = [] }
instances.each do |i|
  managed_by = i.tags.find {|t| t.key == "ManagedBy" }.value
  instances_by_tag[managed_by] << i
end

instances_by_tag.each do |before, is|
  after = before.sub(/\Aspot-autoscaler\//, "spotscaler/")
  params = {
    resources: is.map {|i| i.instance_id },
    tags: [
      {key: 'ManagedBy', value: after},
    ],
  }
  p params
  if execute
    ec2.create_tags(params)
  end
end

resp = ec2.describe_spot_instance_requests(
  filters: [
    {name: "tag:RequestedBy", values: ["spot-autoscaler/*"]},
  ],
)

sirs_by_tag = Hash.new {|h, k| h[k] = [] }
resp.spot_instance_requests.each do |r|
  t = {}
  %w!RequestedBy spot-autoscaler:Status propagate:ManagedBy!.each do |k|
    t[k] = r.tags.find {|t| t.key == k }.value
  end
  sirs_by_tag[t] << r
end

sirs_by_tag.each do |t, sirs|
  params = {
    resources: sirs.map {|sir| sir.spot_instance_request_id },
    tags: [
      {key: 'RequestedBy', value: t['RequestedBy'].sub(/\Aspot-autoscaler\//, "spotscaler/")},
      {key: 'spotscaler:Status', value: t['spot-autoscaler:Status']},
      {key: 'propagate:ManagedBy', value: t['propagate:ManagedBy'].sub(/\Aspot-autoscaler\//, "spotscaler/")},
    ],
  }
  p params
  if execute
    ec2.create_tags(params)
  end
end

unless execute
  puts "To update tags, pass -x"
end
