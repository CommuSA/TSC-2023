import os


def merge_files(filenames, output_filename):
    with open(output_filename, 'wb') as f_out:
        for filename in filenames:
            with open(filename, 'rb') as f_in:
                f_out.write(f_in.read())
                print(f'{filename} has been merged into {output_filename}')


# 设置要合并的文件名列表和输出文件名
filenames = ['kddcup.data_part_50MB', 'kddcup.data_part_600MB']
output_filename = 'kddcup.data_part_650MB'

# 合并文件

merge_files(filenames, output_filename)
