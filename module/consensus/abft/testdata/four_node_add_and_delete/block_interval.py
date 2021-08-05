#  import fileinput
import time
import datetime


def unix_time_millis(dt):
    epoch = datetime.datetime.utcfromtimestamp(0)
    return (dt - epoch).total_seconds() * 1000.0

date_time_str_1 = '2021-04-22 17:44:00.654'
date_time_str_2 = '2021-04-22 17:44:00.401'
date_time_obj_1 = datetime.datetime.strptime(date_time_str_1, '%Y-%m-%d %H:%M:%S.%f')
date_time_obj_2 = datetime.datetime.strptime(date_time_str_2, '%Y-%m-%d %H:%M:%S.%f')
print(unix_time_millis(date_time_obj_1) - unix_time_millis(date_time_obj_1))

print(datetime.datetime.now().timestamp() * 1000)
#  print(date_time_obj_1.time() - date_time_obj_2.time())


#  for line in fileinput.input():
#      print(line)
