import os

filename = "kddcup.part"

filepath = os.path.abspath(filename)

for size in [0.1, 0.2, 0.5, 1, 2, 5, 10, 20]:
    folder_name = f"{int(size * 1024)}KB_files" if size < 1 else f"{int(size)}MB_files"
    if not os.path.exists(folder_name):
        os.mkdir(folder_name)

    with open(filepath, 'rb') as f:
        chunk_size = int(size * 1024 * 1024)
        index = 0
        while True:
            chunk = f.read(chunk_size)
            if not chunk:
                break
            new_filename = f"{folder_name}/part_{index}.part"
            with open(new_filename, 'wb') as new_file:
                new_file.write(chunk)
            index += 1
