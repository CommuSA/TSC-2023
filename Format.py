import re

# 读取文件内容
with open('output.txt', 'r') as f:
    content = f.read()

# 匹配每一行的数据
pattern = r'payload:"\[(.*?)\]"'
matches = re.findall(pattern, content)

# 提取每个数据的数值和单位，并转化为秒
data = []
for match in matches:
    items = match.split(',')
    row = []
    for item in items:
        num, unit = re.findall(r'(\d+\.\d+|\d+)\s*(\w+)', item)[0]
        if unit == 's':
            num = float(num) * 1000
        elif unit == 'ms':
            num = float(num)
        row.append(num)
    data.append(row)

# 输出表格
for row in data:
    print('{:.6f}ms\t{:.6f}ms\t{:.6f}ms'.format(*row))
