-- module
-- Guogeer 2019-08-20 通用的功能

-- module("common",package.seeall)
local common = { _version = "0.0.1" }

function common.print_table(tb,prefix)
	if prefix == nil then
		prefix = ""
	end
	if type(tb) == "table" then
		for k,v in pairs(tb) do
			if type(v) == "table" then
				common.print_table(v,prefix.."\t")
			else
				print(prefix,k,v)
			end
		end
	end
end

function common.in_array(tb, target)
	for k,v in pairs(tb) do
		if v == target then
			return true
		end
	end
	return false
end
-- print(in_array({"A","B"},"B"))

function common.parse_strings(s)
	local parts = {}
	for part in string.gmatch(s,"[^-,;/~]+") do
		table.insert(parts,part)
	end
	return parts
end
-- print_table(parse_strings("10-11001*1,10-11002*2,10-11003*2"))

function common.shuffle(tb,n)
	if n == nil then
		n = #tb	
	end
	if n > #tb then
		n = #tb
	end
	for i=1,n do
		local r = math.random(i,#tb)
		tb[i],tb[r]= tb[r],tb[i]
	end
	return tb
end

--[[
common.print_table(common.shuffle({"A","B","C","D","E","F","G","H","I","J","K","L"},1))
common.print_table(common.shuffle({"A","B","C","D","E","F","G","H","I","J","K","L"},1))
common.print_table(common.shuffle({"A","B","C","D","E","F","G","H","I","J","K","L"}))
common.print_table(common.shuffle({"A","B","C","D","E","F","G","H","I","J","K","L"}))
]]--

function common.rand_sample(samples)
	if #samples == 0 then
		return -1
	end
	
	local sum = 0
	for k,v in pairs(samples) do
		samples[k] = tonumber(v)
	end
	for _,n in pairs(samples) do
		sum = sum + n
	end
	if sum <= 0 then
		return -1
	end

	local t = math.random(sum)
	sum = 0
	for i,n in pairs(samples) do
		sum = sum + n
		if t <= sum then
			return i
		end
	end
	return -1
end
--[[
print(rand_sample({100,1,2,3}))
print(rand_sample({100,1,20000,3}))
]]--

function common.parse_items(s)
	s = string.gsub(s,"*","-")	

	local items = {}
	local values = common.parse_strings(s)
	for i=1,#values,2 do
		local item = {
			["Id"]=tonumber(values[i]),
			["Num"]=tonumber(values[i+1]),
		}
		table.insert(items,item)
	end
	return items
end
-- parse_items("1000*100,1001*123")

function common.split(s,sep)
	local result = {}
	local pattern = "[^" .. sep .. "]+"
	for part in string.gmatch(s, pattern) do
		table.insert(result, part)
	end
	return result
end

-- common.print_table(common.split("a;b;c",";"))

return common
