FROM python:3.15-slim-trixie@sha256:94d3426f1def7db4a106a232ac121bfd975393e5dfd40a619b955fae45fe7913
COPY typos-requirements.txt /tmp/typos-requirements.txt
RUN pip install --no-cache-dir --disable-pip-version-check --require-hashes -r /tmp/typos-requirements.txt \
  && rm /tmp/typos-requirements.txt \
  && groupadd --gid 10001 dev && useradd --uid 10001 --gid 10001 --no-create-home --shell /usr/sbin/nologin dev
USER 10001:10001
WORKDIR /work
ENTRYPOINT ["typos"]
