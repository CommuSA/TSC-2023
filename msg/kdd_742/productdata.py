file_path = 'kddcup.data_part_500MB'
output_sizes = [ 600, 650, 700, 750, 800, 850, 900, 950]

def create_output_file(output_size):
    output_file_path = f'kddcup.data_part_{output_size}MB'
    output_file = open(output_file_path, 'wb')
    return output_file, output_file_path

def merge_files(output_size):
    output_file, output_file_path = create_output_file(output_size)

    with open(file_path, 'rb') as input_file:
        remaining_size = output_size * 1024 * 1024  # Convert MB to bytes
        data = input_file.read(500 * 1024 * 1024)  # Read in chunks of 4KB
        output_file.write(data)

    output_file.close()
    print(f"Merged output file {output_file_path} created.")

for size in output_sizes:
    merge_files(size)