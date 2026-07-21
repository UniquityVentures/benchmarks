from celery import shared_task

@shared_task
def increment_task(val: int) -> int:
    return val + 1
