import os
import shutil

file_path = "msg/kddcup.data"
split_sizes = [0.1, 1, 10, 100]  # MB

split_sizes_bytes = [int(size * 1024 * 1024) for size in split_sizes]

with open(file_path, "rb") as f:
    data = f.read()

    filename, ext = os.path.splitext(file_path)

    for size_bytes in split_sizes_bytes:
        num_files = len(data) // size_bytes + 1

        output_dir = f"{filename}_split_{size_bytes // 1024 // 1024}MB"
        os.makedirs(output_dir, exist_ok=True)

        for i in range(num_files):
            output_filename = f"{output_dir}/part_{i:04}{ext}"
            with open(output_filename, "wb") as output_file:
                output_file.write(data[i * size_bytes:(i + 1) * size_bytes])
